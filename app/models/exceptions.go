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
	ID                int            `db:"id"`
	Calendar_event_id int            `db:"calendar_event_id"`
	StartTimestamp    string         `db:"start_time"`
	EndTimestamp      string         `db:"end_time"`
	Annotation        sql.NullString `db:"annotation"`
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

//TODO: INSERT function (delete slots who collide or keep them?)

/*Validate an exception. */
func (except *Exception) Validate() {
	//TODO
}

/*Update an exception. */
func (except *Exception) Update(v *revel.Validation) (err error) {
	//TODO
	return
}

const (
	stmtSelectExeptions = `
    SELECT id, calendar_event_id, start_time, end_time, annotation
    FROM calendar_exceptions
    WHERE start_time BETWEEN ($1) AND ($2)
		ORDER BY start_time ASC
  `
)
