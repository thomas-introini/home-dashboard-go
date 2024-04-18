package routes

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"it.introini/home-dashboard/db"
	"it.introini/home-dashboard/globals"
	"it.introini/home-dashboard/templates"
)

const (
	TS_FORMAT    = "Monday Jan 2 15:04:05"
	CHART_FORMAT = "02/01 15:04"
)

var log = globals.Logger()

func getLimitOffset(r http.Request) (limit int, offset int) {
	params := r.URL.Query()
	if params.Get("limit") == "" {
		limit = 10
	} else {
		limit, _ = strconv.Atoi(params.Get("limit"))
	}
	if params.Get("offset") == "" {
		offset = 0
	} else {
		offset, _ = strconv.Atoi(params.Get("offset"))
	}
	return
}

func getChartRequestParams(r http.Request) (period time.Duration, interval time.Duration) {
	params := r.URL.Query()
	if params.Get("period") == "" {
		period = 7 * 24 * time.Hour
	} else {
		period, _ = time.ParseDuration(params.Get("period"))
	}
	if params.Get("interval") == "" {
		interval = time.Hour

	} else {
		interval, _ = time.ParseDuration(params.Get("interval"))
	}
	return period, interval
}

func getIfNoneMatchHeader(r http.Request) string {
	header := r.Header.Get("If-None-Match")
	header, _ = strings.CutPrefix(header, "W/\"")
	header, _ = strings.CutSuffix(header, "\"")
	return header
}

func getSensorData(limit int, offset int) ([]templates.SensorData, time.Time, error) {
	data, err := db.GetSensorData(limit, offset)
	if err != nil {
		return nil, time.Time{}, err
	}
	sensorRows := make([]templates.SensorData, 0)
	for _, row := range data {
		sensorRows = append(sensorRows, templates.SensorData{
			Id:          strconv.Itoa(row.Id),
			Temperature: row.Temperature,
			Humidity:    row.Humidity,
			Date:        row.Date.In(globals.GetTz()).Format(TS_FORMAT),
		})
	}

	return sensorRows, data[0].Date, nil
}

func getChartData(period time.Duration, interval time.Duration) (templates.SensorChartData, error) {
	from := time.Now().Add(-period)
	to := time.Now()
	groupedData, err := db.GetGroupedData(from, to, interval)
	chartData := templates.SensorChartData{
		Labels:            make([]string, len(groupedData)),
		TemperatureSeries: make([]float64, len(groupedData)),
		HumiditySeries:    make([]float64, len(groupedData)),
	}
	if err != nil {
		return chartData, err
	}
	for i, row := range groupedData {
		chartData.Labels[i] = row.Date.In(globals.GetTz()).Format(CHART_FORMAT)
		chartData.TemperatureSeries[i] = row.Temperature
		chartData.HumiditySeries[i] = row.Humidity
	}
	return chartData, nil
}

func getETag(updatedOn time.Time) string {
	return fmt.Sprintf(
		"W/\"%s\"",
		updatedOn.Format(time.RFC3339),
	)
}

func RootHandler(w http.ResponseWriter, r *http.Request) {
	limit, offset := getLimitOffset(*r)
	period, interval := getChartRequestParams(*r)
	etag := getIfNoneMatchHeader(*r)
	if etag != "" {
		clientUpdatedOn, err := time.Parse(time.RFC3339, etag)
		if err == nil { // Ignore cache if cannot parse
			serverUpdatedOn, err := db.GetLastUpdatedOn()
			if err == nil && serverUpdatedOn == clientUpdatedOn {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}
	}
	data, updatedOn, err := getSensorData(limit, offset)
	if err != nil {
		log.Error("could not get sensor data %s\n", err)
		http.Error(w, err.Error(), 500)
		return
	}
	log.Info("Found rows", slog.Int("rows", len(data)))
	chartData, err := getChartData(period, interval)
	if err != nil {
		fmt.Printf("could not get group %s\n", err)
		http.Error(w, err.Error(), 500)
		return
	}
	log.Info("Found grouped rows", slog.Int("rows", len(chartData.Labels)))

	c := templates.RootPage(data, chartData, limit, offset, period, interval)
	w.Header().Add("ETag", getETag(updatedOn))
	log.Info("ETag", "etag", w.Header().Get("ETag"))
	c.Render(context.Background(), w)
}

func GetMoreRowsHandler(w http.ResponseWriter, r *http.Request) {
	limit, offset := getLimitOffset(*r)
	data, _, err := getSensorData(limit, offset)
	if err != nil {
		log.Error("Could not get sensor data", err)
		http.Error(w, err.Error(), 500)
		return
	}
	log.Info("Got rows", slog.Int("rows", len(data)))
	rows := templates.SensorDataRows(data, true)
	btn := templates.LoadMoreDataButton(limit, offset+limit)
	c := templates.Composite(rows, btn)

	c.Render(context.Background(), w)
}

func GetSensorChartHandler(w http.ResponseWriter, r *http.Request) {
	period, interval := getChartRequestParams(*r)
	chartData, err := getChartData(period, interval)
	if err != nil {
		fmt.Printf("could not get group %s\n", err)
		http.Error(w, err.Error(), 500)
		return
	}
	fmt.Printf("Found %d grouped rows\n", len(chartData.Labels))
	chart := templates.SensorChart(chartData, period, interval)
	chart.Render(context.Background(), w)
}

type InsertSensorDataBody struct {
	Temperature float64 `json:"temperature,omitempty"`
	Humidity    float64 `json:"humidity,omitempty"`
}

func InsertSensorDataHandler(w http.ResponseWriter, r *http.Request) {
	temperatureStr := r.URL.Query().Get("temperature")
	humidityStr := r.URL.Query().Get("humidity")
	if temperatureStr == "" {
		http.Error(w, "temperature is required", 400)
		return
	}
	if humidityStr == "" {
		http.Error(w, "humidity is required", 400)
		return
	}

	temperature, err := strconv.ParseFloat(temperatureStr, 64)
	if err != nil {
		http.Error(w, "temperature must be a number", 400)
		return
	}
	humidity, err := strconv.ParseFloat(humidityStr, 64)
	if err != nil {
		http.Error(w, "humidity must be a number", 400)
		return
	}

	log.Info("data", "temperature", temperature, "humidity", humidity)
	err = db.InsertSensorData(time.Now(), temperature, humidity)
	if err != nil {
		log.Error("could not insert data", err)
		http.Error(w, "could not insert data", 500)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "{\"message\":\"Data stored correctly\"}")
}
