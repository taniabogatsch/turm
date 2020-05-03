package models

import (
	"database/sql"
	"turm/app"

	"github.com/jmoiron/sqlx"
	"github.com/revel/revel"
)

/*Event is a model of the event table. */
type Event struct {
	ID            int            `db:"id, primarykey, autoincrement"`
	CourseID      int            `db:"courseid"`
	Capacity      int            `db:"capacity"`
	HasWaitlist   bool           `db:"haswaitlist"`
	Title         string         `db:"title"`
	Annotation    sql.NullString `db:"annotation"`
	EnrollmentKey sql.NullString `db:"enrollmentkey"`
	Meetings      Meetings       ``

	//Fullness is the number of users that enrolled in this event
	Fullness int ``
	//Percentage is (Fullness * 100) / Capacity
	Percentage int ``
}

/*Validate event fields. */
func (event *Event) Validate(v *revel.Validation) {
	//TODO
}

/*NewBlank creates a new blank event. */
func (event *Event) NewBlank() (err error) {

	err = app.Db.Get(event, stmtInsertBlankEvent, event.CourseID, event.Title)
	if err != nil {
		modelsLog.Error("failed to insert blank event", "event", event,
			"error", err.Error())
	}
	return
}

/*Update the specified column in the event table. */
func (event *Event) Update(column string, value interface{}) (err error) {
	return updateByID(column, value, event.ID, "event", event)
}

/*Delete an event. */
func (event *Event) Delete() (err error) {

	_, err = app.Db.Exec(stmtDeleteEvent, event.ID)
	if err != nil {
		modelsLog.Error("failed to delete event", "eventID", event.ID,
			"error", err.Error())
	}
	return
}

/*Events holds all events of a course. */
type Events []Event

/*Get all events of a course. */
func (events *Events) Get(tx *sqlx.Tx, courseID *int) (err error) {

	err = tx.Select(events, stmtSelectEvents, *courseID)
	if err != nil {
		modelsLog.Error("failed to get events of course", "course ID", *courseID, "error", err.Error())
		tx.Rollback()
		return
	}

	//get all meetings of this event
	for key := range *events {
		(*events)[key].Percentage = ((*events)[key].Fullness * 100) / (*events)[key].Capacity
		if err = (*events)[key].Meetings.Get(tx, &(*events)[key].ID); err != nil {
			return
		}
	}
	return
}

const (
	stmtSelectEvents = `
		SELECT
			e.id, e.courseid, e.capacity, e.haswaitlist,
			e.title, e.annotation, e.enrollmentkey,
			(
				SELECT COUNT(en.userid)
				FROM enrolled en
				WHERE en.eventid = e.id
					AND status != 1 /*on waitlist*/
			) AS fullness
		FROM event e
		WHERE courseid = $1
		ORDER BY id ASC
	`

	stmtInsertBlankEvent = `
		INSERT INTO event (
				courseid, title, capacity, haswaitlist
			)
		VALUES (
				$1, $2, 1, false
		)
		RETURNING id, title
	`

	stmtDeleteEvent = `
		DELETE FROM event
		WHERE id = $1
	`
)
