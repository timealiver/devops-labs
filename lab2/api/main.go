package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	requests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests.",
	}, []string{"method", "path", "status"})
	errorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_errors_total",
		Help: "Total HTTP responses with status 5xx.",
	}, []string{"path"})
	duration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency.",
		Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 3, 5},
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

func instrument(path string, next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next(sw, r)
		elapsed := time.Since(start)
		requests.WithLabelValues(r.Method, path, strconv.Itoa(sw.status)).Inc()
		duration.WithLabelValues(path).Observe(elapsed.Seconds())
		if sw.status >= 500 {
			errorsTotal.WithLabelValues(path).Inc()
		}
		slog.InfoContext(r.Context(), "request",
			"path", path,
			"status", sw.status,
			"duration_ms", elapsed.Milliseconds(),
			"trace_id", trace.SpanContextFromContext(r.Context()).TraceID().String(),
		)
	})
}

func health(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "ok")
}

func fail(w http.ResponseWriter, r *http.Request) {
	span := trace.SpanFromContext(r.Context())
	err := errors.New("simulated failure")
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	slog.ErrorContext(r.Context(), "handler failed",
		"error", err.Error(),
		"trace_id", span.SpanContext().TraceID().String(),
	)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func slow(w http.ResponseWriter, r *http.Request) {
	_, span := otel.Tracer("api").Start(r.Context(), "slow-op")
	time.Sleep(time.Duration(1000+rand.Intn(2000)) * time.Millisecond)
	span.End()
	fmt.Fprint(w, "done")
}

func load(client *http.Client, port string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		n := 50
		if v, err := strconv.Atoi(r.URL.Query().Get("n")); err == nil && v > 0 {
			n = v
		}
		var wg sync.WaitGroup
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				req, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, "http://localhost:"+port+"/health", nil)
				resp, err := client.Do(req)
				if err == nil {
					resp.Body.Close()
				}
			}()
		}
		wg.Wait()
		fmt.Fprintf(w, "sent %d\n", n)
	}
}

func initTracing(ctx context.Context) (func(context.Context) error, error) {
	exporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, err
	}
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewSchemaless(semconv.ServiceName("api"))),
	)
	otel.SetTracerProvider(provider)
	return provider.Shutdown, nil
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	prometheus.MustRegister(requests, errorsTotal, duration)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	shutdown, err := initTracing(ctx)
	if err != nil {
		slog.Error("tracing init failed", "error", err.Error())
		os.Exit(1)
	}

	client := &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
	mux := http.NewServeMux()
	mux.Handle("/health", instrument("/health", health))
	mux.Handle("/fail", instrument("/fail", fail))
	mux.Handle("/slow", instrument("/slow", slow))
	mux.Handle("/load", instrument("/load", load(client, port)))
	mux.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{Addr: ":" + port, Handler: otelhttp.NewHandler(mux, "api")}
	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(sctx)
		shutdown(sctx)
	}()

	slog.Info("listening", "addr", ":"+port)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server stopped", "error", err.Error())
		os.Exit(1)
	}
}
