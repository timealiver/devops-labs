package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	pool     *pgxpool.Pool
	requests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "shop_api_requests_total",
		Help: "API requests by path and status.",
	}, []string{"path", "status"})
	latency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "shop_api_request_duration_seconds",
		Help:    "API request latency.",
		Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2},
	}, []string{"path"})
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func count(path string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next(sw, r)
		requests.WithLabelValues(path, fmt.Sprint(sw.status)).Inc()
		latency.WithLabelValues(path).Observe(time.Since(start).Seconds())
	}
}

func health(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("HEALTH_FAIL") == "true" || pool == nil {
		http.Error(w, "unhealthy", http.StatusServiceUnavailable)
		return
	}
	fmt.Fprint(w, "ok")
}

func version(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, os.Getenv("VERSION"))
}

func order(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	item := r.URL.Query().Get("item")
	if item == "" {
		var body struct{ Item string }
		json.NewDecoder(r.Body).Decode(&body)
		item = body.Item
	}
	if item == "" {
		item = "unnamed"
	}
	if pool == nil {
		http.Error(w, "db not ready", http.StatusServiceUnavailable)
		return
	}
	var id int
	err := pool.QueryRow(r.Context(), "INSERT INTO orders(item) VALUES($1) RETURNING id", item).Scan(&id)
	if err != nil {
		slog.Error("insert failed", "error", err.Error())
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"id": id, "item": item})
}

func orders(w http.ResponseWriter, r *http.Request) {
	if pool == nil {
		http.Error(w, "db not ready", http.StatusServiceUnavailable)
		return
	}
	rows, err := pool.Query(r.Context(), "SELECT id, item, status FROM orders ORDER BY id DESC LIMIT 50")
	if err != nil {
		slog.Error("select failed", "error", err.Error())
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var id int
		var item, status string
		if err := rows.Scan(&id, &item, &status); err != nil {
			break
		}
		list = append(list, map[string]any{"id": id, "item": item, "status": status})
	}
	json.NewEncoder(w).Encode(list)
}

func pendingOrders() float64 {
	if pool == nil {
		return 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var n int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM orders WHERE status='new'").Scan(&n); err != nil {
		return 0
	}
	return float64(n)
}

func connect(ctx context.Context) {
	for {
		p, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
		if err == nil {
			_, err = p.Exec(ctx, "CREATE TABLE IF NOT EXISTS orders(id serial PRIMARY KEY, item text NOT NULL, status text NOT NULL DEFAULT 'new', created_at timestamptz DEFAULT now())")
			if err == nil {
				pool = p
				return
			}
			p.Close()
		}
		slog.Warn("waiting for database", "error", err.Error())
		time.Sleep(2 * time.Second)
	}
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	prometheus.MustRegister(requests, latency, prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "shop_orders_pending",
		Help: "Orders waiting for the worker.",
	}, pendingOrders))
	mux := http.NewServeMux()
	mux.HandleFunc("/health", count("/health", health))
	mux.HandleFunc("/version", version)
	mux.HandleFunc("/order", count("/order", order))
	mux.HandleFunc("/orders", count("/orders", orders))
	mux.Handle("/metrics", promhttp.Handler())
	go connect(context.Background())
	slog.Info("listening", "version", os.Getenv("VERSION"))
	if err := http.ListenAndServe(":8080", mux); err != nil {
		slog.Error("server stopped", "error", err.Error())
		os.Exit(1)
	}
}
