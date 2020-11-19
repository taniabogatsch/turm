package models

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/revel/revel"
)

/*ExceptionsOfWeek holds all exeptions of a week [0....6]. */
type ExceptionsOfWeek []Exceptions

/*Exceptions locked at a specific day within StartTime and EndTime of a day template. */
type Exceptions []Exception

/*Exception can lock a timespan on a specific date. */
type Exception struct {
	ID              int            `db:"id"`
	CalendarEventID int            `db:"calendar_event_id"`
	ExceptionStart  string         `db:"exception_start"`
	ExceptionEnd    string         `db:"exception_end"`
	Annotation      sql.NullString `db:"annotation"`

	//used to get the front end values
	ExceptionStartTime string ``
	ExceptionEndTime   string ``
}

/*Get all exceptions of a day . Monday specifies the week for which all exceptions
must be loaded and weekday specifies the day. */
func (excepts *Exceptions) Get(tx *sqlx.Tx, monday time.Time, weekday int) (err error) {

	//[startTime, endTime] is [day 00:00 - day 24:00]
	startTime := monday.AddDate(0, 0, weekday-1)
	endTime := startTime.Add(1000000000 * 60 * 60 * 24)

	err = tx.Select(excepts, stmtSelectExeptions, startTime, endTime)
	if err != nil {
		log.Error("failed to get exceptions of day template", "monday", monday,
			"weekday", weekday, "error", err.Error())
		tx.Rollback()
	}
	return
}

//validate an exception.
func (except *Exception) validate() {
	//TODO
}

/*Insert an exception. */
func (except *Exception) Insert(v *revel.Validation) (err error) {
	//TODO: INSERT function (delete slots who collide or keep them?)
	//TODO
	return
}

/*Update an exception. */
func (except *Exception) Update(v *revel.Validation) (err error) {
	//TODO
	return
}

const (
	stmtSelectExeptions = `
    SELECT id, calendar_event_id, exception_start, exception_end, annotation
    FROM calendar_exceptions
    WHERE exception_start BETWEEN ($1) AND ($2)
		ORDER BY exception_start ASC
  `
)
