// Command hyperblur runs hyperblur web server.
package main

import (
	"log"
	"net/http"
	"os"
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
		IdleTimeout:       90 * time.Second,
	}

	logger.Println("Starting up hyperblur")
	if err := server.ListenAndServe(); err != nil {
		logger.Fatal(err)
	}
}
