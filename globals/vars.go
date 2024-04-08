package globals

import (
	"log/slog"
	"time"
)

var Tz *time.Location
var log *slog.Logger = slog.Default()

func LoadTz() error {
	tz, err := time.LoadLocation("Europe/Rome")
	if err != nil {
		return err
	}
	Tz = tz
	return nil
}

func GetTz() *time.Location {
	return Tz
}

func Logger() *slog.Logger {
	return log
}
