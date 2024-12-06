package fast_utils

import (
	"time"
)

const (
	TIME_YYYY_MM_DD_HH_MM_SS_SSS = "2006-01-02 15:04:05.000"
	TIME_YYYY_MM_DD_HH_MM_SS     = "2006-01-02 15:04:05"
	TIME_YYYY_MM_DD              = "2006-01-02"
)

func GetTimeStr(t *time.Time) string {
	if t == nil {
		return time.Now().Format(TIME_YYYY_MM_DD_HH_MM_SS)
	}
	return t.Format(TIME_YYYY_MM_DD_HH_MM_SS)
}

func GetTimeSSSStr(t *time.Time) string {
	if t == nil {
		return time.Now().Format(TIME_YYYY_MM_DD_HH_MM_SS)
	}
	return t.Format(TIME_YYYY_MM_DD_HH_MM_SS_SSS)
}

func GetDateStr(t *time.Time) string {
	if t == nil {
		return time.Now().Format(TIME_YYYY_MM_DD)
	}
	return t.Format(TIME_YYYY_MM_DD)
}

func ToTime(timeStr string) time.Time {
	t, _ := time.ParseInLocation(TIME_YYYY_MM_DD_HH_MM_SS, timeStr, time.Local)
	return t
}
