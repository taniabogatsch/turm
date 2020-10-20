package models

import (
	"strconv"
	"strings"
)

type Custom_time struct {
	Value string
	Hour  int
	Min   int
}

/*SetTime uses a Timestring of "12:50". if it is a valid time it sets hour and min*/
func (t1 *Custom_time) SetTime(value string) (succsess bool) {

	s := strings.Split(value, ":")

	h, error1 := strconv.Atoi(s[0])
	m, error2 := strconv.Atoi(s[1])

	if error1 != nil || error2 != nil {
		log.Error("on SetTime: not convertable to int",
			"error_hour", error1, "error_min", error2)
	} else {
		if 0 <= h && h < 24 && 0 <= m && m < 60 {
			succsess = true
			t1.Value = value
			t1.Hour = h
			t1.Min = m
		}
	}
	return
}

/*Before checks if a time(t1) is before a secound time(t2)*/
func (t1 *Custom_time) Before(t2 Custom_time) (isBefore bool) {
	if t1.Hour < t2.Hour {
		isBefore = true
		return
	} else if t1.Hour == t2.Hour {
		if t1.Min < t2.Min {
			isBefore = true
			return
		}
	}
	isBefore = false
	return
}

/*After checks if a time(t1) is after a secound time(t2)*/
func (t1 *Custom_time) After(t2 Custom_time) (isBefore bool) {
	if t1.Hour > t2.Hour {
		isBefore = true
		return
	} else if t1.Hour == t2.Hour {
		if t1.Min > t2.Min {
			isBefore = true
			return
		}
	}
	isBefore = false
	return
}

/*Sub gives the difference from two times within a day*/
func (t1 *Custom_time) Sub(t2 Custom_time) (min int) {
	if t1.After(t2) {
		min = (t1.Hour-t2.Hour)*60 + (t1.Min - t2.Min)
	} else {
		min = (t2.Hour-t1.Hour)*60 + (t2.Min - t1.Min)
	}
	return
}
