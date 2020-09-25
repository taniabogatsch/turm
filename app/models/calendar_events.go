package models

import (
	"database/sql"
	"turm/app"
)

/*CalendarEvent is a special calendar-based event of a course. */
type CalendarEvent struct {
	ID          int            `db:"id"`
	CourseID    int            `db:"course_id"`
	Title       string         `db:"title"`
	Annotations sql.NullString `db:"annotations"`
	Created     string         `db:"created"`
	Creator     sql.NullInt64  `db:"creator"`

	//loaded week
	Week int
	//day templates for this week
	Days []DayTmpls
}

/*DayTmpls of a specific day. */
type DayTmpls []DayTmpl

/*DayTmpl is a section of a week day (Monday - Sunday). */
type DayTmpl struct {
	ID              int    `db:"id"`
	CalendarEventID int    `db:"calendar_event_id"`
	StartTime       string `db:"start_time"`
	EndTime         string `db:"end_time"`
	Intervall       int    `db:"intervall"`
	//must be an integer between [1,7]
	DayOfWeek        int            `db:"day_of_week"`
	Active           bool           `db:"active"`
	DeactiavtionDate sql.NullString `db:"deactivation_date"`
	Created          string         `db:"created"`
	Creator          sql.NullInt64  `db:"creator"`

	Slots      Slots
	Exceptions Exceptions
}

/*Slots booked at a specific day within StartTime and EndTime of a day template. */
type Slots []Slot

/*Slot is a booked timespan on an specific date. */
type Slot struct {
	ID        int    `db:"id"`
	DayTmplID int    `db:"day_tmpl_id"`
	UserID    int    `db:"user_id"`
	Created   string `db:"created"`

	//date + time
	StartTimestamp string `db:"start_time"`
	EndTimestamp   string `db:"end_time"`
}

/*Exceptions locked at a specific day within StartTime and EndTime of a day template. */
type Exceptions []Exception

/*Exception can lock a timespan on a specific date. */
type Exception struct {
	ID        int `db:"id"`
	DayTmplID int `db:"day_tmpl_id"`
	//date + time
	StartTimestamp string         `db:"start_time"`
	EndTimestamp   string         `db:"end_time"`
	Annotations    sql.NullString `db:"annotations"`
	Created        string         `db:"created"`
}

/*NewBlank creates a new blank CalendarEvent. */
func (event *CalendarEvent) NewBlank() (err error) {

	err = app.Db.Get(event, stmtInsertCalendarEvent, event.CourseID, event.Title)
	if err != nil {
		log.Error("failed to insert blank calendar event", "event", *event,
			"error", err.Error())
	}
	return
}

func (event *CalendarEvent) EditTitle() (err error) {
	err = app.Db.Get(event, stmtEditCalendarEventTitle, event.CourseID, event.Title)
	if err != nil {
		log.Error("failed to insert blank calendar event", "event", *event,
			"error", err.Error())
	}
	return
}

/*EditAnnotation of an Callendar*/
func (event *CalendarEvent) EditAnnotation() (err error) {
	err = app.Db.Get(event, stmtEditCalendarEventAnnotation, event.CourseID, event.Annotations)
	if err != nil {
		log.Error("failed to insert blank calendar event", "event", *event,
			"error", err.Error())
	}
	return
}

/*Insert a DayTemplate. */
func (dayTmpl *DayTmpl) Insert() (err error) {

	//TODO: update fields + statement
	err = app.Db.Get(dayTmpl, stmtInsertDayTemplate, dayTmpl.CalendarEventID, dayTmpl.StartTime,
		dayTmpl.EndTime, dayTmpl.Intervall, dayTmpl.DayOfWeek)
	if err != nil {
		log.Error("failed to insert day template", "dayTmpl", *dayTmpl,
			"error", err.Error())
	}
	return
}

/*NewSlot inserts a new slot and creates corresponding day if not existent already*/
func (slot *Slot) NewSlot(date string) (err error) {
	// check if day is already created: n -> create new
	// check if slot is free: y -> insert  n -> return error
	return
}

const (
	stmtInsertCalendarEvent = `
		INSERT INTO calendar_events (
				course_id, title
			)
		VALUES (
				$1, $2
		)
		RETURNING id
	`

	stmtInsertDayTemplate = `
		INSERT INTO day_templates (
				calendar_event_id, start_time, end_time, intervall, day_of_week
			)
		VALUES (
				$1, $2, $3, $4, $5
		)
		RETURNING id
	`

	//use Update function
	stmtEditCalendarEventTitle = `
	UPDATE calendar_events
		SET title = $2
	WHERE id =$1;`

	stmtEditCalendarEventAnnotation = `
	UPDATE calendar_events
		SET annotations = $2
	WHERE id =$1;`
)
