package tgo

import (
	"fmt"
	"time"
)

func UtilTimeGetDate(t time.Time) time.Time {

	year, month, day := t.Date()

	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

func UtilTimeIsToday(t time.Time) bool {
	return UtilTimeSameDay(t, time.Now())
}

func UtilTimeSameDay(t1 time.Time, t2 time.Time) bool {
	if UtilTimeDiffDay(t1, t2) == 0 {
		return true
	}
	return false
}

func UtilTimeDiffDay(t1 time.Time, t2 time.Time) int {
	return int(UtilTimeGetDate(t2).Sub(UtilTimeGetDate(t1)) / (24 * time.Hour))
}

func UtilTimeFewDaysLater(day int) time.Time {
	return UtilTimeFewDurationLater(time.Duration(day) * 24 * time.Hour)
}

func UtilTimeTwentyFourHoursLater() time.Time {
	return UtilTimeFewDurationLater(time.Duration(24) * time.Hour)
}

func UtilTimeSixHoursLater() time.Time {
	return UtilTimeFewDurationLater(time.Duration(6) * time.Hour)
}

func UtilTimeFewDurationLater(duration time.Duration) time.Time {
	baseTime := time.Now()
	fewDurationLater := baseTime.Add(duration)
	return fewDurationLater
}

func UtilTimeIsExpired(expirationTime time.Time) bool {
	after := time.Now().After(expirationTime)
	return after
}

//UtilTimeConsumePrint print time consume
func UtilTimeConsumePrint(timeBefore time.Time, prefix string) {
	duration := time.Now().Sub(timeBefore)
	fmt.Printf("%s time :%d\n", prefix, duration.Nanoseconds()/1000000)
}

func UtilTimeIsLeapYear(y int) bool {
	if (y%400 == 0) || (y%4 == 0 && y%100 != 0) {
		return true
	} else {
		return false
	}
}
