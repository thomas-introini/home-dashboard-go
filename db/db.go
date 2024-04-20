package db

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"it.introini/home-dashboard/globals"
)

var DB *sql.DB

var DB_FORMAT = "2006-01-02 15:04:05"

type SensorData struct {
	Id          int
	Temperature float64
	Humidity    float64
	Date        time.Time
}

func ConnectDB() error {
	db, err := sql.Open("sqlite3", "./sensor.db")
	if err != nil {
		return err
	}
	_, err = os.Stat("./sensor.db")
	if os.IsNotExist(err) {
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS sensor (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				temperature INTEGER,
				humidity INTEGER,
				date DATETIME DEFAULT CURRENT_TIMESTAMP
			)`)
		if err != nil {
			return err
		}
	}
	_, err = db.Exec("PRAGMA busy_timeout = 5000;")
	if err != nil {
		globals.Logger().Error("Error setting busy timeout: %s", err)
		return nil
	}
	DB = db
	return nil
}

func GetSensorData(limit int, offset int) ([]SensorData, error) {
	rows, err := DB.Query("SELECT * FROM sensor ORDER BY date DESC LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return nil, err
	}
	keep_going := rows.Next()
	data := make([]SensorData, 0)
	for keep_going {
		var (
			id          int
			temperature float64
			humidity    float64
			date        time.Time
		)
		err := rows.Scan(&id, &temperature, &humidity, &date)
		if err != nil {
			return nil, err
		}
		data = append(data, SensorData{id, temperature, humidity, date})
		keep_going = rows.Next()
	}

	return data, nil
}

func GetGroupedData(from time.Time, to time.Time, interval time.Duration) ([]SensorData, error) {
	fmt.Printf("from %s to %s interval %f\n", from, to, interval.Seconds())
	rows, err := DB.Query(`
		SELECT datetime(strftime('%s', date) / ? * ?, 'unixepoch') as d,
			   AVG(temperature) as temperature,
			   AVG(humidity) as humidity
		  FROM sensor
		 WHERE date BETWEEN ? AND ?
		 GROUP BY 1
		 ORDER BY 1`,
		int(interval.Seconds()), int(interval.Seconds()), from, to)

	if err != nil {
		return nil, err
	}

	next := rows.Next()
	data := make([]SensorData, 0)
	for next {
		var (
			dateStr     string
			temperature float64
			humidity    float64
		)
		err := rows.Scan(&dateStr, &temperature, &humidity)
		if err != nil {
			return nil, err
		}
		if dateStr == "" {
			continue
		}
		date, err := time.Parse(DB_FORMAT, dateStr)
		if err != nil {
			return nil, err
		}
		data = append(data, SensorData{Temperature: temperature, Humidity: humidity, Date: date})
		next = rows.Next()
	}

	return data, nil
}

func InsertSensorData(now time.Time, temperature float64, humidity float64) error {
	_, err := DB.Exec("INSERT INTO sensor (date, temperature, humidity) VALUES (?, ?, ?)", now.Format(DB_FORMAT), temperature, humidity)
	return err
}

func GetLastUpdatedOn() (time.Time, error) {
	rows, err := DB.Query("SELECT MAX(date) as max FROM sensor")
	if err != nil {
		return time.Time{}, err
	}

	next := rows.Next()
	if !next {
		return time.Time{}, err
	}

	var dateStr string
	err = rows.Scan(&dateStr)
	date, err := time.Parse(DB_FORMAT, dateStr)
	if err != nil {
		return time.Time{}, err
	}
	return date, nil
}
