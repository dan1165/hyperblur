// Command hyperblur runs the hyperblur web server.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/dan1165/hyperblur/web"
)

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags)

	port := os.Getenv("HYPERBLUR_PORT")
	if port == "" {
		port = "8000"
	}

	app := web.NewApp(logger)

	logger.Println("Starting up hyperblur")
	if err := http.ListenAndServe(":"+port, app.Handler()); err != nil {
		logger.Fatal(err)
	}
}
