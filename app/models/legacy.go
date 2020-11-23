package models

import (
	"strconv"
	"strings"
	"time"
)

/*OldCourse is the old struct of a course. */
type OldCourse struct {
	CourseName             string
	Subtitle               string
	CourseLeader           []OldUserList
	Limitation             []OldLimitations
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
	Event                  []OldEvent
	Blacklist              []OldUserList
	Whitelist              []OldUserList
	Parent                 string
	Prioid                 int
	EnrollLimitEvents      int
	WelcomeMail            string
}

/*OldEvent is the old struct of an event. */
type OldEvent struct {
	WaitingList         string
	Description         string
	MaximumParticipants string
	EnrollmentKey1      string
	EnrollmentKey2      string
	InitEnrollmentKey   string
	Meeting             []OldMeeting
}

/*OldMeeting is the old struct of a meeting. */
type OldMeeting struct {
	MeetingRegularity string
	Day               string
	WeeklyInterval    string
	MeetingDate       string
	MeetingStartTime  string
	MeetingEndTime    string
	Location          string
	Annotation        string
}

/*OldLimitations is the old struct used for course restrictions. */
type OldLimitations struct {
	OnlyLDAP        string
	Degree          string
	CourseOfStudies string
	Semester        string
}

/*OldUserList is the old struct used for user lists. */
type OldUserList struct {
	Uid       int
	Firstname string
	Lastname  string
	Email     string
}

/*Transform an old course struct to the new course struct. */
func (oldCourse *OldCourse) Transform(course *Course) (err error) {

	if oldCourse.Subtitle != "" {
		course.Subtitle.Valid = true
		course.Subtitle.String = oldCourse.Subtitle
	}

	course.Visible, err = strconv.ParseBool(oldCourse.Public)
	if err != nil {
		log.Error("failed to transform visible flag", "public", oldCourse.Public,
			"error", err.Error())
		return
	}

	if oldCourse.Description != "" {
		course.Description.Valid = true
		course.Description.String = oldCourse.Description
	}

	if oldCourse.PaymentAmount != "0" {
		oldCourse.PaymentAmount = strings.ReplaceAll(oldCourse.PaymentAmount, ",", ".")
		course.Fee.Valid = true
		course.Fee.Float64, err = strconv.ParseFloat(oldCourse.PaymentAmount, 64)
		if err != nil {
			log.Error("failed to transform fee", "payment amount", oldCourse.PaymentAmount,
				"error", err.Error())
			return
		}
	}

	if oldCourse.WelcomeMail != "" && oldCourse.WelcomeMail != "<p>&nbsp;</p>" {
		course.CustomEMail.Valid = true
		course.CustomEMail.String = oldCourse.WelcomeMail
	}

	if oldCourse.EnrollLimitEvents != 0 {
		course.EnrollLimitEvents.Valid = true
		course.EnrollLimitEvents.Int32 = int32(oldCourse.EnrollLimitEvents)
	}

	course.EnrollmentStart = oldCourse.EnrollmentStartDate + " " + oldCourse.EnrollmentStartTime
	course.EnrollmentEnd = oldCourse.EnrollmentEndDate + " " + oldCourse.EnrollmentEndTime
	course.ExpirationDate = oldCourse.ExpirationDate + " " + oldCourse.ExpirationTime

	if oldCourse.DisenrollmentStartDate != "" {
		course.UnsubscribeEnd.Valid = true
		course.UnsubscribeEnd.String = oldCourse.DisenrollmentStartDate + " " + oldCourse.DisenrollmentStartTime
	}

	//get onlyLDAP
	for _, limitation := range oldCourse.Limitation {
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
	for i, oldEvent := range oldCourse.Event {
		event := Event{}
		if err = oldEvent.Transform(&event, i); err != nil {
			return
		}
		course.Events = append(course.Events, event)
	}

	return
}

/*Transform an old event struct to the new event struct. */
func (oldEvent *OldEvent) Transform(event *Event, i int) (err error) {

	event.Capacity, err = strconv.Atoi(oldEvent.MaximumParticipants)

	event.HasWaitlist, err = strconv.ParseBool(oldEvent.WaitingList)
	if err != nil {
		log.Error("failed to transform has waitlist flag", "waitingList", oldEvent.WaitingList,
			"error", err.Error())
		return
	}

	if oldEvent.Description != "" {
		event.Title = oldEvent.Description
	} else {
		event.Title = "Imported event " + strconv.Itoa(i+1)
	}

	//transform all meetings
	for _, oldMeeting := range oldEvent.Meeting {
		meeting := Meeting{}
		oldMeeting.Transform(&meeting)
		event.Meetings = append(event.Meetings, meeting)
	}

	return
}

/*Transform an old meeting struct to the new meeting struct. */
func (oldMeeting *OldMeeting) Transform(meeting *Meeting) {

	if oldMeeting.MeetingRegularity == "periodic" {

		//get the interval
		switch oldMeeting.WeeklyInterval {
		case "everyWeek":
			meeting.MeetingInterval = WEEKLY
		case "evenWeek":
			meeting.MeetingInterval = EVEN
		case "oddWeek":
			meeting.MeetingInterval = ODD
		}

		//get the week day
		meeting.WeekDay.Valid = true
		switch oldMeeting.Day {
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

	if oldMeeting.Location != "" {
		meeting.Place.Valid = true
		meeting.Place.String = oldMeeting.Location
	}

	if oldMeeting.Annotation != "" {
		meeting.Annotation.Valid = true
		meeting.Annotation.String = oldMeeting.Annotation
	}

	if meeting.MeetingInterval != SINGLE {
		oldMeeting.MeetingDate = time.Now().Format("2006-01-02")
	}
	meeting.MeetingStart = oldMeeting.MeetingDate + " " + oldMeeting.MeetingStartTime
	meeting.MeetingEnd = oldMeeting.MeetingDate + " " + oldMeeting.MeetingEndTime
}
