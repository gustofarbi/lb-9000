package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	counter := &atomic.Int32{}
	go logger(counter)

	router := http.NewServeMux()
	router.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		counter.Add(1)
		defer counter.Add(-1)

		sleepTime := rand.Intn(8)
		time.Sleep(time.Duration(sleepTime) * time.Second)

		if _, err := w.Write([]byte(
			fmt.Sprintf(
				"%s: slept for %d seconds at %s",
				os.Getenv("POD_NAME"),
				sleepTime,
				r.URL.Path,
			))); err != nil {
			slog.Error("error writing response", "err", err)
		}
	}))
	server := http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	waiter := make(chan struct{}, 1)
	go signalHandler(&server, waiter)

	go func() {
		if err := server.ListenAndServe(); err != nil {
			slog.Error("error listening", "err", err)
		}
	}()
	<-waiter
}

func logger(counter *atomic.Int32) {
	for {
		slog.Info("counter", "count", counter.Load())
		time.Sleep(1 * time.Second)
	}
}

func signalHandler(server *http.Server, waiter chan struct{}) {
	sigint := make(chan os.Signal, 1)

	signal.Notify(sigint, os.Interrupt)
	signal.Notify(sigint, syscall.SIGTERM)

	<-sigint

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("error shutting down server", "err", err)
	}
	waiter <- struct{}{}
	close(waiter)
}
