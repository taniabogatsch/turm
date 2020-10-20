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

	//loaded week
	Week int

	//day templates for this week [0...6]
	Days DayTmpls
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

/*Get all CalendarEvents. */
func (events *CalendarEvents) Get(tx *sqlx.Tx, courseID *int, day time.Time) (err error) {

	txWasNil := (tx == nil)
	if txWasNil {
		tx, err = app.Db.Beginx()
		if err != nil {
			log.Error("failed to begin tx", "error", err.Error())
			return
		}
	}

	err = tx.Select(events, stmtSelectCalendarEvents, *courseID)
	if err != nil {
		log.Error("failed to get CalendarEvents of course", "course ID", *courseID,
			"error", err.Error())
		tx.Rollback()
		return
	}

	for i := range *events {
		//get all day_templates of this event
		err = (*events)[i].Days.Get(tx, &(*events)[i].ID, day)
		if err != nil {
			return
		}
		//set the current week
		_, (*events)[i].Week = day.ISOWeek()
	}

	if txWasNil {
		tx.Commit()
	}
	return
}

/*Get a calendar event by its ID and a monday. */
func (event *CalendarEvent) Get(courseID *int, monday time.Time) (err error) {
	//TODO
	return
}

/*Update the specific column in the CalendarEvent. */
func (event *CalendarEvent) Update(column string, value interface{}) (err error) {
	return updateByID(nil, column, "calendar_events", value, event.ID, event)
}

/*Delete a calendar event. */
func (event *CalendarEvent) Delete(v *revel.Validation) (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	var notEmpty bool
	//TODO @Marco:
	//don't allow the deletion of calendar events if users are enrolled in them

	if notEmpty {
		v.ErrorKey("validation.invalid.delete")
		tx.Commit()
		return
	}

	//delete event
	if err = deleteByID("id", "calendar_events", event.ID, tx); err != nil {
		return
	}

	tx.Commit()
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

	stmtSelectCalendarEvents = `
		SELECT id, course_id, title, annotation
		FROM calendar_events
		WHERE course_id = $1
		ORDER BY id ASC
	`

	stmtGetCourseIDByCalendarEvent = `
		SELECT course_id
		FROM calendar_events
		WHERE id = $1
	`
)
