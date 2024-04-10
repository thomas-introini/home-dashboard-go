package routes

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
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

func getSensorData(limit int, offset int) ([]templates.SensorData, error) {
	data, err := db.GetSensorData(limit, offset)
	if err != nil {
		return nil, err
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

	return sensorRows, nil
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

func RootHandler(w http.ResponseWriter, r *http.Request) {
	limit, offset := getLimitOffset(*r)
	period, interval := getChartRequestParams(*r)
	data, err := getSensorData(limit, offset)
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
	c.Render(context.Background(), w)
}

func GetMoreRowsHandler(w http.ResponseWriter, r *http.Request) {
	limit, offset := getLimitOffset(*r)
	data, err := getSensorData(limit, offset)
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

}
