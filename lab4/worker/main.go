package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	processed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "shop_orders_processed_total",
		Help: "Orders marked as done by the worker.",
	})
)

func process(ctx context.Context, pool *pgxpool.Pool) {
	tag, err := pool.Exec(ctx, "UPDATE orders SET status='done' WHERE id IN (SELECT id FROM orders WHERE status='new' ORDER BY id LIMIT 10 FOR UPDATE SKIP LOCKED)")
	if err != nil {
		slog.Warn("process failed", "error", err.Error())
		return
	}
	if n := tag.RowsAffected(); n > 0 {
		processed.Add(float64(n))
		slog.Info("processed orders", "count", n)
	}
}

func run(ctx context.Context) {
	for {
		pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
		if err == nil {
			_, err = pool.Exec(ctx, "CREATE TABLE IF NOT EXISTS orders(id serial PRIMARY KEY, item text NOT NULL, status text NOT NULL DEFAULT 'new', created_at timestamptz DEFAULT now())")
		}
		if err != nil {
			slog.Warn("waiting for database", "error", err.Error())
			time.Sleep(2 * time.Second)
			continue
		}
		for {
			process(ctx, pool)
			time.Sleep(2 * time.Second)
		}
	}
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "ok") })
	mux.Handle("/metrics", promhttp.Handler())
	go run(context.Background())
	slog.Info("worker listening")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		slog.Error("server stopped", "error", err.Error())
		os.Exit(1)
	}
}
