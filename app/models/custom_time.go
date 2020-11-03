package models

import (
	"strconv"
	"strings"
)

/*CustomTime is used to compare times. */
type CustomTime struct {
	Value string
	Hour  int
	Min   int
}

/*SetTime uses a time string of format '12:50'. If value is a valid time it sets hour and min. */
func (t1 *CustomTime) SetTime(value string) (success bool) {

	s := strings.Split(value, ":")

	h, err := strconv.Atoi(s[0])
	if err != nil {
		log.Error("cannot convert value to hour", "value", value,
			"error", err.Error())
		return
	}

	m, err := strconv.Atoi(s[1])
	if err != nil {
		log.Error("cannot convert value to minute", "value", value,
			"error", err.Error())
		return
	}

	if 0 <= h && h <= 24 && 0 <= m && m < 60 {
		success = true
		t1.Value = value
		t1.Hour = h
		t1.Min = m
	}
	return
}

/*Before checks if t1 is before t2. */
func (t1 *CustomTime) Before(t2 *CustomTime) (before bool) {

	if t1.Hour < t2.Hour {
		before = true
		return
	}

	if t1.Hour == t2.Hour && t1.Min <= t2.Min {
		before = true
	}
	return
}

/*After checks if t1 is after t2. */
func (t1 *CustomTime) After(t2 *CustomTime) (after bool) {
	return !t1.Before(t2)
}

/*Sub returns the time interval between two times. */
func (t1 *CustomTime) Sub(t2 *CustomTime) (min int) {

	if t1.After(t2) {
		min = (t1.Hour-t2.Hour)*60 + (t1.Min - t2.Min)
	} else {
		min = (t2.Hour-t1.Hour)*60 + (t2.Min - t1.Min)
	}
	return
}

/*Equals checks if t1 equals t2. */
func (t1 *CustomTime) Equals(t2 *CustomTime) (equals bool) {

	if t1.Hour == t2.Hour && t1.Min == t2.Min {
		return true
	}
	return
}

/*String sets the value field of the CustomTime struct. */
func (t1 *CustomTime) String() {

	h := strconv.Itoa(t1.Hour)
	m := strconv.Itoa(t1.Min)
	t1.Value = h + ":" + m
}
