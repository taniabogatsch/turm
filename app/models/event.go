package models

import (
	"database/sql"

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
	Description   sql.NullString `db:"description"`
	EnrollmentKey sql.NullString `db:"enrollmentkey"`
	Meetings      Meetings       ``
}

/*Validate event fields. */
func (event *Event) Validate(v *revel.Validation) {
	//TODO
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
		if err = (*events)[key].Meetings.Get(tx, &(*events)[key].ID); err != nil {
			return
		}
	}
	return
}

const (
	stmtSelectEvents = `
		SELECT
			id, courseid, capacity, haswaitlist, title, description, enrollmentkey
		FROM event
		WHERE courseid = $1
		ORDER BY id ASC
	`
)
