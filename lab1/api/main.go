package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
)

var (
	mu    sync.Mutex
	hoard [][]byte
)

func health(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "ok")
}

func eat(w http.ResponseWriter, r *http.Request) {
	mb, err := strconv.Atoi(r.URL.Query().Get("mb"))
	if err != nil || mb < 0 {
		http.Error(w, "usage: /eat?mb=N", http.StatusBadRequest)
		return
	}
	chunk := make([]byte, mb<<20)
	for i := 0; i < len(chunk); i += 4096 {
		chunk[i] = 1
	}
	mu.Lock()
	hoard = append(hoard, chunk)
	total := 0
	for _, c := range hoard {
		total += len(c)
	}
	mu.Unlock()
	fmt.Fprintf(w, "holding %d MB\n", total>>20)
}

func burn(w http.ResponseWriter, r *http.Request) {
	go func() {
		for {
		}
	}()
	fmt.Fprintln(w, "burning one core")
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.HandleFunc("/health", health)
	http.HandleFunc("/eat", eat)
	http.HandleFunc("/burn", burn)
	log.Printf("listening on :%s (pid=%d, uid=%d)", port, os.Getpid(), os.Getuid())
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
