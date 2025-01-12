package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
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
	router.Handle("/", handle(counter))

	server := http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 30 * time.Second,
	}

	waiter := make(chan struct{}, 1)
	go signalHandler(&server, waiter)

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("error listening", "err", err)
		}
	}()
	<-waiter
}

func logger(counter *atomic.Int32) {
	for {
		slog.Info("counter", "count", counter.Load())
		time.Sleep(5 * time.Second)
	}
}

func signalHandler(server *http.Server, waiter chan struct{}) {
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-sigint

	slog.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("error shutting down server", "err", err)
	}
	waiter <- struct{}{}
	close(waiter)
}

func handle(counter *atomic.Int32) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		counter.Add(1)
		defer counter.Add(-1)

		sleepTime, err := rand.Int(rand.Reader, big.NewInt(8))
		if err != nil {
			http.Error(w, "cannot get random number", http.StatusInternalServerError)
			return
		}

		time.Sleep(time.Duration(sleepTime.Int64()) * time.Second)

		if _, err = fmt.Fprintf(
			w,
			"%s: slept for %d seconds at %s",
			os.Getenv("POD_NAME"),
			sleepTime,
			r.URL.Path,
		); err != nil {
			slog.Error("error writing response", "err", err)
		}
	}
}
