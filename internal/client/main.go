package main

import (
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"strconv"
)

var (
	semaphore = func() chan struct{} {
		numWorkers := os.Getenv("NUM_WORKERS")
		if numWorkers == "" {
			numWorkers = "30"
		}

		n, err := strconv.Atoi(numWorkers)
		if err != nil {
			n = 30
		}

		return make(chan struct{}, n)
	}()
	acquire = func() { semaphore <- struct{}{} }
	release = func() { <-semaphore }
)

func main() {
	service := "lb-service"
	port := 8080

	for {
		url := fmt.Sprintf(
			"http://%s:%d/%s",
			service,
			port,
			randString(10),
		)
		acquire()
		go func() {
			defer release()
			fetchURL(url)
		}()
	}
}

func fetchURL(url string) {
	response, err := http.Get(url)
	if err != nil {
		slog.Error("error getting pod", "err", err)
		return
	}

	if response.StatusCode == http.StatusOK {
		b, err := io.ReadAll(response.Body)
		if err != nil {
			slog.Error("error reading response body", "err", err)
		}
		slog.Info("response", "body", string(b))
	}
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
