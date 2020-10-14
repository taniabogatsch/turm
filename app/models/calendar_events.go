package models

import (
	"database/sql"
	"time"
	"turm/app"

	"github.com/jmoiron/sqlx"
	"github.com/revel/revel"
)

/*CalendarEvents holds all calendar events of a course. */
type CalendarEvents []CalendarEvent

/*CalendarEvent is a special calendar-based event of a course. */
type CalendarEvent struct {
	ID         int            `db:"id"`
	CourseID   int            `db:"course_id"`
	Title      string         `db:"title"`
	Annotation sql.NullString `db:"annotation"`
	Created    string         `db:"created"`
	Creator    sql.NullInt64  `db:"creator"`

	//loaded week
	Week int
	//day templates for this week
	Days DayTmpls
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
	StartTimestamp time.Time `db:"start_time"`
	EndTimestamp   time.Time `db:"end_time"`
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

/*Get all CalendarEvents*/
func (events *CalendarEvents) Get(tx *sqlx.Tx, courseID *int, day time.Time) (err error) {

	err = tx.Select(events, stmtSelectCalendarEvents, *courseID)
	if err != nil {
		log.Error("failed to get CalendarEvents of course", "course ID", *courseID, "error", err.Error())
		tx.Rollback()
		return
	}

	for i := range *events {
		//get all day_templates of this event

		(*events)[i].Days.Get(tx, &(*events)[i].ID, day)

	}
	return
}

/*Update the specific column in the CalendarEvent. */
func (event *CalendarEvent) Update(column string, value interface{}) (err error) {
	return updateByID(nil, column, "calendar_events", value, event.ID, event)
}

/*Delete a calendar event. */
func (event *CalendarEvent) Delete() (err error) {
	//TODO
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

/*Get all dayTmpls of a CalendarEvent*/
func (dayTmpls *DayTmpls) Get(tx *sqlx.Tx, calendarEventID *int, monday time.Time) (err error) {

	err = tx.Select(dayTmpls, stmtSelectDayTmpls, *calendarEventID)
	if err != nil {
		log.Error("failed to get DayTemplates of course", "calendarEventID", *calendarEventID, "error", err.Error())
		tx.Rollback()
		return
	}

	//@TODO Marco: get exeptions & slots
	for i := range *dayTmpls {
		(*dayTmpls)[i].Slots.Get(tx, (*dayTmpls)[i].ID, monday, (*dayTmpls)[i].DayOfWeek)
		(*dayTmpls)[i].Exceptions.Get(tx, monday, (*dayTmpls)[i].DayOfWeek)
	}
	return
}

/*NewSlot inserts a new slot and creates corresponding day if not existent already*/
func (slot *Slot) NewSlot(date string) (err error) {
	//ckeck if time is within a day template
	// check if slot is free: y -> insert  n -> return error
	return
}

/*Get the slots with a dayTmplID and who are in this day*/
func (slots *Slots) Get(tx *sqlx.Tx, dayTmplID int, day time.Time, weekday int) (err error) {

	//calculate start of and end timestamp for this day for db query
	startTime := day.AddDate(0, 0, weekday-1)
	endTime := startTime.Add(1000000000 * 60 * 60 * 24) //24h

	err = tx.Select(slots, stmtSelectSlots, dayTmplID, startTime, endTime)
	if err != nil {
		log.Error("failed to get slots of dayTemplate", "DayTmplID", dayTmplID, "error", err.Error())
		tx.Rollback()
		return
	}
	return
}

/*Validate the startTime and endTime of a slot  */
func (slot *Slot) Validate(v *revel.Validation) {

	startInPast := slot.StartTimestamp.After(time.Now())
	v.Check(startInPast).MessageKey("validation.calendarEvent.startInPast")

	startAfterEndTime := slot.StartTimestamp.Before(slot.EndTimestamp)
	v.Check(startAfterEndTime).MessageKey("validation.calendarEvent.startAfterEndTime")

	dayTmpls := []DayTmpl{}

	weekday := int(slot.StartTimestamp.Weekday())

	err := app.Db.Select(dayTmpls, stmtGetDayTemplateFromWeekDay, weekday)
	if err != nil {
		log.Error("failed to get DayTemolate", "weekday", weekday,
			"error", err.Error())
	}

	slotStartTime := CustomTime{}
	slotStartTime.Hour, slotStartTime.Min, _ = slot.StartTimestamp.Clock()

	slotEndTime := CustomTime{}
	slotEndTime.Hour, slotEndTime.Min, _ = slot.EndTimestamp.Clock()

	tmplStartTime := CustomTime{}
	tmplEndTime := CustomTime{}

	isInTemplate := false
	indexDayTmpl := 0

	//look for a DayTemplate where the slot is within its time intervall
	for i := range dayTmpls {
		tmplStartTime.SetTime(dayTmpls[i].StartTime)
		tmplEndTime.SetTime(dayTmpls[i].EndTime)

		if tmplStartTime.Before(slotStartTime) && tmplEndTime.After(slotEndTime) {
			isInTemplate = true
			indexDayTmpl = i
			break
		}
	}

	v.Check(isInTemplate).MessageKey("validation.calendarEvent.noTemplateFitting")

	//schrittweite prüfen
	//types ok?
	intervallSteps := float64(slotStartTime.Sub(tmplStartTime) / dayTmpls[indexDayTmpl].Intervall)
	startWrongStepDistance := (intervallSteps - float64(int(intervallSteps))) == 0
	v.Check(startWrongStepDistance).MessageKey("validation.calendarEvent.startTimeWrongStepDistance")

	intervallSteps = float64(slotEndTime.Sub(tmplStartTime) / dayTmpls[indexDayTmpl].Intervall)
	endWrongStepDistance := (intervallSteps - float64(int(intervallSteps))) == 0
	v.Check(endWrongStepDistance).MessageKey("validation.calendarEvent.endTimeWrongStepDistance")

	//schon besetzt?
}

/*Get all exeptions for a weekday following a specific monday */
func (exepts *Exceptions) Get(tx *sqlx.Tx, day time.Time, weekday int) (err error) {

	//calculate start of and end timestamp for this day for db query
	startTime := day.AddDate(0, 0, weekday-1)
	endTime := startTime.Add(1000000000 * 60 * 60 * 24) //24h

	err = tx.Select(exepts, stmtSelectExeptions, startTime, endTime)
	if err != nil {
		log.Error("failed to get slots of dayTemplate", "error", err.Error())
		tx.Rollback()
		return
	}

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

	stmtGetDayTemplateFromWeekDay = `
	SELECT id, start_time, end_time, intevall
	FROM day_templates
	WHERE day_of_week = $1
		AND active = true
	`

	stmtSelectCalendarEvents = `
		SELECT id, course_id, title, annotations, created, creator_id
		FROM calendar_events
		WHERE course_id = $1
	`

	stmtSelectDayTmpls = `
		SELECT id, calendar_event_id, start_time, end_time, intervall, day_of_week, created, creator_id, active, deactivation_date
		FROM day_templates
		WHERE calendar_event_id = $1 AND active = true
	`

	stmtSelectSlots = `
	SELECT id, user_id, day_tmpl_id, start_time, end_time, created
	FROM slots
	WHERE day_tmpl_id = $1 AND start_time BETWEEN ($2) AND ($3);
	`

	stmtSelectExeptions = `
	SELECT id, calendar_event_id, start_time, end_time, annotations, created
	FROM calendar_exceptions
	WHERE start_time BETWEEN ($1) AND ($2);
	`
)
