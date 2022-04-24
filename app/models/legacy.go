package models

import (
	"strconv"
	"strings"
	"time"
)

/*Version2Course is the version 2 struct of a course. */
type Version2Course struct {
	CourseName             string
	Subtitle               string
	CourseLeader           []Version2UserList
	Limitation             []Version2Limitations
	EnrollmentStartDate    string
	EnrollmentEndDate      string
	EnrollmentStartTime    string
	EnrollmentEndTime      string
	DisenrollmentStartDate string
	DisenrollmentStartTime string
	ExpirationDate         string
	ExpirationTime         string
	Public                 string
	PaymentAmount          string
	Description            string
	Event                  []Version2Event
	Blacklist              []Version2UserList
	Whitelist              []Version2UserList
	Parent                 string
	Prioid                 int
	EnrollLimitEvents      int
	WelcomeMail            string
}

/*Version2Event is the version 2 struct of an event. */
type Version2Event struct {
	WaitingList         string
	Description         string
	MaximumParticipants string
	EnrollmentKey1      string
	EnrollmentKey2      string
	InitEnrollmentKey   string
	Meeting             []Version2Meeting
}

/*Version2Meeting is the version 2 struct of a meeting. */
type Version2Meeting struct {
	MeetingRegularity string
	Day               string
	WeeklyInterval    string
	MeetingDate       string
	MeetingStartTime  string
	MeetingEndTime    string
	Location          string
	Annotation        string
}

/*Version2Limitations is the version 2 struct used for course restrictions. */
type Version2Limitations struct {
	OnlyLDAP        string
	Degree          string
	CourseOfStudies string
	Semester        string
}

/*Version2UserList is the version 2 struct used for user lists. */
type Version2UserList struct {
	Uid       int
	Firstname string
	Lastname  string
	Email     string
}

/*Version3Course contains the Blacklist and Whitelist field for backwards compatibility. */
type Version3Course struct {
	Course
	Blacklist UserList ``
	Whitelist UserList ``
}

/*Transform a version 2 course struct to the current course struct. */
func (version2Course *Version2Course) Transform(course *Course) (err error) {

	//NOTE: user lists are not transformed because user IDs of the version 2
	//system do not match the ones of the current system

	if version2Course.Subtitle != "" {
		course.Subtitle.Valid = true
		course.Subtitle.String = version2Course.Subtitle
	}

	course.Visible, err = strconv.ParseBool(version2Course.Public)
	if err != nil {
		log.Error("failed to transform visible flag", "public", version2Course.Public,
			"error", err.Error())
		return
	}

	if version2Course.Description != "" {
		course.Description.Valid = true
		course.Description.String = version2Course.Description
	}

	if version2Course.PaymentAmount != "0" {
		version2Course.PaymentAmount = strings.ReplaceAll(version2Course.PaymentAmount, ",", ".")
		course.Fee.Valid = true
		course.Fee.Float64, err = strconv.ParseFloat(version2Course.PaymentAmount, 64)
		if err != nil {
			log.Error("failed to transform fee", "payment amount", version2Course.PaymentAmount,
				"error", err.Error())
			return
		}
	}

	if version2Course.WelcomeMail != "" && version2Course.WelcomeMail != "<p>&nbsp;</p>" {
		course.CustomEMail.Valid = true
		course.CustomEMail.String = version2Course.WelcomeMail
	}

	if version2Course.EnrollLimitEvents != 0 {
		course.EnrollLimitEvents.Valid = true
		course.EnrollLimitEvents.Int32 = int32(version2Course.EnrollLimitEvents)
	}

	enrollStart, err := getTimestamp(version2Course.EnrollmentStartDate + " " + version2Course.EnrollmentStartTime)
	if err != nil {
		return
	}
	course.EnrollmentStart = enrollStart

	enrollEnd, err := getTimestamp(version2Course.EnrollmentEndDate + " " + version2Course.EnrollmentEndTime)
	if err != nil {
		return
	}
	course.EnrollmentEnd = enrollEnd

	expirationDate, err := getTimestamp(version2Course.ExpirationDate + " " + version2Course.ExpirationTime)
	if err != nil {
		return
	}
	course.ExpirationDate = expirationDate

	if version2Course.DisenrollmentStartDate != "" {
		unsubscribeEnd, err := getTimestamp(version2Course.DisenrollmentStartDate + " " + version2Course.DisenrollmentStartTime)
		if err != nil {
			return err
		}
		course.UnsubscribeEnd.Valid = true
		course.UnsubscribeEnd.Time = unsubscribeEnd
	}

	//get onlyLDAP
	for _, limitation := range version2Course.Limitation {
		onlyLDAP, err := strconv.ParseBool(limitation.OnlyLDAP)
		if err != nil {
			log.Error("failed to transform only ldap flag", "onlyLDAP", limitation.OnlyLDAP,
				"error", err.Error())
			return err
		}
		if onlyLDAP {
			course.OnlyLDAP = true
		}
	}

	//transform all events
	for i, oldEvent := range version2Course.Event {
		event := Event{}
		if err = oldEvent.Transform(&event, i); err != nil {
			return
		}
		course.Events = append(course.Events, event)
	}

	return
}

/*Transform a version 2 event struct to the current event struct. */
func (version2Event *Version2Event) Transform(event *Event, i int) (err error) {

	event.Capacity, err = strconv.Atoi(version2Event.MaximumParticipants)

	event.HasWaitlist, err = strconv.ParseBool(version2Event.WaitingList)
	if err != nil {
		log.Error("failed to transform has waitlist flag", "waitingList", version2Event.WaitingList,
			"error", err.Error())
		return
	}

	if version2Event.Description != "" {
		event.Title = version2Event.Description
	} else {
		event.Title = "Imported event " + strconv.Itoa(i+1)
	}

	//transform all meetings
	for _, version2Meeting := range version2Event.Meeting {
		meeting := Meeting{}
		if err = version2Meeting.Transform(&meeting); err != nil {
			return
		}
		event.Meetings = append(event.Meetings, meeting)
	}

	return
}

/*Transform a version 2 meeting struct to the current meeting struct. */
func (version2Meeting *Version2Meeting) Transform(meeting *Meeting) (err error) {

	if version2Meeting.MeetingRegularity == "periodic" {

		//get the interval
		switch version2Meeting.WeeklyInterval {
		case "everyWeek":
			meeting.MeetingInterval = WEEKLY
		case "evenWeek":
			meeting.MeetingInterval = EVEN
		case "oddWeek":
			meeting.MeetingInterval = ODD
		}

		//get the week day
		meeting.WeekDay.Valid = true
		switch version2Meeting.Day {
		case "Monday":
			meeting.WeekDay.Int32 = 0
		case "Tuesday":
			meeting.WeekDay.Int32 = 1
		case "Wednesday":
			meeting.WeekDay.Int32 = 2
		case "Thursday":
			meeting.WeekDay.Int32 = 3
		case "Friday":
			meeting.WeekDay.Int32 = 4
		case "Saturday":
			meeting.WeekDay.Int32 = 5
		case "Sunday":
			meeting.WeekDay.Int32 = 6
		}

	} else {
		meeting.MeetingInterval = SINGLE
	}

	if version2Meeting.Location != "" {
		meeting.Place.Valid = true
		meeting.Place.String = version2Meeting.Location
	}

	if version2Meeting.Annotation != "" {
		meeting.Annotation.Valid = true
		meeting.Annotation.String = version2Meeting.Annotation
	}

	if meeting.MeetingInterval != SINGLE {
		version2Meeting.MeetingDate = time.Now().Format("2006-01-02")
	}

	start, err := getTimestamp(version2Meeting.MeetingDate + " " + version2Meeting.MeetingStartTime)
	if err != nil {
		return
	}
	meeting.MeetingStart = start

	end, err := getTimestamp(version2Meeting.MeetingDate + " " + version2Meeting.MeetingEndTime)
	if err != nil {
		return
	}
	meeting.MeetingEnd = end

	return
}

/*Transform a version 3 course struct to the current course struct. */
func (version3Course *Version3Course) Transform(course *Course) {
	course.Blocklist = version3Course.Blacklist
	course.Allowlist = version3Course.Whitelist
}
