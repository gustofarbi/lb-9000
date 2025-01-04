package main

import (
	"fmt"
	"github.com/google/uuid"
	"io"
	"log/slog"
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
			uuid.New(),
		)

		acquire()
		go func() {
			defer release()
			fetchURL(url)
		}()
	}
}

var client = &http.Client{}

func fetchURL(url string) {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		slog.Error("cannot create request", "error", err)
		return
	}
	response, err := client.Do(request)
	if err != nil {
		slog.Error("error getting pod", "err", err)
		return
	}

	if response.StatusCode == http.StatusOK {
		b, err := io.ReadAll(response.Body)
		if err != nil {
			slog.Error("error reading response body", "err", err)
			return
		}

		slog.Info("response", "body", string(b))
	}
}
