package models

import (
	"strconv"
	"strings"
)

type CustomTime struct {
	Value string
	Hour  int
	Min   int
}

/*SetTime uses a Timestring of "12:50". if it is a valid time it sets hour and min*/
func (time *CustomTime) SetTime(value string) (succsess bool) {

	s := strings.Split(value, ":")

	h, error1 := strconv.Atoi(s[0])
	m, error2 := strconv.Atoi(s[1])

	if error1 != nil || error2 != nil {
		log.Error("on SetTime: hour is not convertable to int",
			"error_hour", error1, "error_min", error2)
	}

	if error1 == nil && error2 == nil {
		if 0 <= h && h < 24 && 0 <= m && m < 60 {
			succsess = true
			time.Value = value
			time.Hour = h
			time.Min = m
		}
	}
	return
}
