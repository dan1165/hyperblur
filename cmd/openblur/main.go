// Command openblur runs the openblur web server.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/dan1165/openblur/web"
)

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags)

	assetsDir := os.Getenv("OPENBLUR_ASSETS_DIR")
	if assetsDir == "" {
		assetsDir = "assets"
	}

	port := os.Getenv("OPENBLUR_PORT")
	if port == "" {
		port = "8000"
	}

	app := web.NewApp(logger)

	logger.Println("Starting up openblur")
	if err := http.ListenAndServe(":"+port, app.Handler(assetsDir)); err != nil {
		logger.Fatal(err)
	}
}
