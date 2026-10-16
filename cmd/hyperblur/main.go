// Command hyperblur runs hyperblur web server.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dan1165/hyperblur/web"
)

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags)

	port := os.Getenv("HYPERBLUR_PORT")
	if port == "" {
		port = "8000"
	}

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
