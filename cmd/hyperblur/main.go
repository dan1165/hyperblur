// Command hyperblur runs hyperblur web server.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dan1165/hyperblur/web"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-healthcheck", "--healthcheck":
			// Docker healthcheck for the scratch image, which has no shell or
			// curl to probe with.
			os.Exit(healthcheck())
		}
	}

	logger := log.New(os.Stdout, "", log.LstdFlags)

	port := envPort()

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           web.NewApp(logger).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       90 * time.Second,
		MaxHeaderBytes:    1 << 16,
	}

	// No WriteTimeout: media is streamed, and a deadline would cut long
	// downloads. Stop accepting work on SIGINT/SIGTERM and drain instead.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Printf("Starting up hyperblur %s on :%s", web.Version, port)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatalf("hyperblur stopped: %v", err)
		}
	}()

	<-ctx.Done()
	stop()
	logger.Print("Shutting down hyperblur")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Printf("shutdown error: %v", err)
	}
}

func envPort() string {
	if port := os.Getenv("HYPERBLUR_PORT"); port != "" {
		return port
	}
	return "8000"
}

// healthcheck requests the local /healthz endpoint and reports it as an exit
// code, for use as a Docker HEALTHCHECK command.
func healthcheck() int {
	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Get("http://127.0.0.1:" + envPort() + "/healthz")
	if err != nil {
		fmt.Fprintf(os.Stderr, "healthcheck: %v\n", err)
		return 1
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "healthcheck: HTTP %d\n", response.StatusCode)
		return 1
	}
	return 0
}
