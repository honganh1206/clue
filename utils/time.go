package utils

import (
	_ "embed"
	"fmt"
	"time"
)

var timeFormats = []string{
	time.RFC3339Nano,                      // 2006-01-02T15:04:05.999999999Z07:00
	time.RFC3339,                          // 2006-01-02T15:04:05Z07:00
	"2006-01-02 15:04:05",                 // SQLite default format
	"2006-01-02 15:04:05.999999999-07:00", // SQLite with nanoseconds and offset
}

func ParseTimeWithFallback(timeStr string) (time.Time, error) {
	for _, format := range timeFormats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse time '%s' with any known format", timeStr)
}
