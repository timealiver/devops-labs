package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"
)

func envInt(name string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(name)); err == nil {
		return v
	}
	return def
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	mb := envInt("BATCH_MEM_MB", 50)
	cpus := envInt("BATCH_CPU", 1)

	buf := make([]byte, mb<<20)
	go func() {
		for {
			for i := 0; i < len(buf); i += 4096 {
				buf[i]++
			}
			time.Sleep(time.Second)
		}
	}()

	var sink atomic.Uint64
	for i := 0; i < cpus; i++ {
		go func() {
			for {
				sink.Add(1)
			}
		}()
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "ok") })
	slog.Info("batch started", "mem_mb", mb, "cpu", cpus)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		slog.Error("server stopped", "error", err.Error())
		os.Exit(1)
	}
}
