package main

import (
	"errors"
	"log/slog"
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
	log.Info("Starting Home dashdoard...")
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

	handler := NewServer(log)

	httpServer := http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	log.Info("Listening on :8080")
	if err = httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("error starting server", "error", err)
	}

	if errors.Is(err, http.ErrServerClosed) {
		log.Error("Server closed")
	} else if err != nil {
		log.Error("Server closed", "error", err)
		os.Exit(1)
	}
}

func NewServer(log *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	staticFolder := os.Getenv("STATIC_FOLDER")
	if staticFolder == "" {
		staticFolder = "./static"
	}
	log.Info("folder " + staticFolder)
	fs := http.FileServer(http.Dir(staticFolder))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	addRoutes(mux)

	return mux
}

func addRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /{$}", routes.RootHandler)
	mux.HandleFunc("GET /api/get-more-rows", routes.GetMoreRowsHandler)
	mux.HandleFunc("GET /api/get-sensor-chart", routes.GetSensorChartHandler)
	mux.HandleFunc("POST /api/data", routes.InsertSensorDataHandler)
}
