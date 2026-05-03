package util

import "time"

func NowText() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func DayText(t time.Time) string {
	return t.Format("2006-01-02")
}
