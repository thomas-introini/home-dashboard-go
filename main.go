package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"it.introini/home-dashboard/db"
	"it.introini/home-dashboard/globals"
	"it.introini/home-dashboard/routes"
)

const (
	TS_FORMAT    = "Monday Jan 2 15:04:05"
	CHART_FORMAT = "02/01 15:04"
)

func main() {
	log := globals.Logger()
	err := globals.LoadTz()
	if err != nil {
		log.Error("Could not load TZ", "error", err)
		os.Exit(1)
	}

	err = db.ConnectDB()
	if err != nil {
		log.Error("Could not connect to DB", "error", err)
		os.Exit(1)
	}

	handler := NewServer()

	httpServer := http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	log.Info("Listening on :8080")
	if err = httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("error starting server", "error", err)
	}

	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}

func NewServer() http.Handler {
	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	addRoutes(mux)

	return mux
}

func addRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /{$}", routes.RootHandler)
	mux.HandleFunc("GET /api/get-more-rows", routes.GetMoreRowsHandler)
	mux.HandleFunc("GET /api/get-sensor-chart", routes.GetSensorChartHandler)
}
