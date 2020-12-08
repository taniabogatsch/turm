package models

import (
	"database/sql"
	"strconv"
	"strings"
	"time"
	"turm/app"

	"github.com/jmoiron/sqlx"
	"github.com/revel/revel"
)

/*ExceptionsOfWeek holds all exeptions of a week [0....6]. */
type ExceptionsOfWeek []Exception

/*Exception can lock a timespan on a specific date. */
type Exception struct {
	ID               int            `db:"id"`
	CalendarEventID  int            `db:"calendar_event_id"`
	ExceptionStartDB time.Time      `db:"exception_start"`
	ExceptionEndDB   time.Time      `db:"exception_end"`
	Annotation       sql.NullString `db:"annotation"`

	//used to get the front end values
	ExceptionStart     string `db:"exception_start_str"`
	ExceptionEnd       string `db:"exception_end_str"`
	ExceptionStartTime string ``
	ExceptionEndTime   string ``
}

/*Exceptions is a slice of exceptions. */
type Exceptions []Exception

/*Get all exceptions of a week. Monday specifies the week for which all exceptions
must be loaded. */
func (excepts *ExceptionsOfWeek) Get(tx *sqlx.Tx, eventID *int, monday time.Time) (err error) {

	monday = time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, monday.Location())
	endTime := monday.AddDate(0, 0, 7)

	err = tx.Select(excepts, stmtSelectExceptionsOfWeek, eventID, monday, endTime)
	if err != nil {
		log.Error("failed to get exceptions of week", "monday", monday,
			"endTime", endTime, "error", err.Error())
		tx.Rollback()
	}
	return
}

//validate an exception
func (except *Exception) validate(v *revel.Validation, tx *sqlx.Tx) (err error) {

	//get the time values
	startTime := CustomTime{}
	endTime := CustomTime{}

	if except.Annotation.String != "" {

		except.Annotation.String = strings.TrimSpace(except.Annotation.String)
		v.Check(except.Annotation.String,
			revel.MinSize{3},
			revel.MaxSize{255},
		).MessageKey("validation.invalid.text.short")

		except.Annotation.Valid = true
	}

	if !startTime.SetTime(except.ExceptionStartTime) {
		v.ErrorKey("validation.calendar.exceptions.invalid.time")
		return
	}

	if !endTime.SetTime(except.ExceptionEndTime) {
		v.ErrorKey("validation.calendar.exceptions.invalid.time")
		return
	}

	//get dates out of string for timestamp creation
	location, err := time.LoadLocation(app.TimeZone)
	if err != nil {
		log.Error("failed to load location", "appTimeZone", app.TimeZone,
			"error", err.Error())
		tx.Rollback()
		return
	}

	splitStartDate := strings.Split(except.ExceptionStart, "-")

	yearStart, err := strconv.Atoi(splitStartDate[0])
	if err != nil {
		log.Error("failed to convert string to int", "splitStartDate[0]",
			splitStartDate[0], "error", err.Error())
		tx.Rollback()
		return
	}

	monthStart, err := strconv.Atoi(splitStartDate[1])
	if err != nil {
		log.Error("failed to convert string to int", "splitStartDate[1]",
			splitStartDate[1], "error", err.Error())
		tx.Rollback()
		return
	}

	dayStart, err := strconv.Atoi(splitStartDate[2])
	if err != nil {
		log.Error("failed to convert string to int", "splitStartDate[2]",
			splitStartDate[2], "error", err.Error())
		tx.Rollback()
		return
	}

	splitEndDate := strings.Split(except.ExceptionEnd, "-")

	yearEnd, err := strconv.Atoi(splitEndDate[0])
	if err != nil {
		log.Error("failed to convert string to int", "splitEndDate[0]",
			splitEndDate[0], "error", err.Error())
		tx.Rollback()
		return
	}

	monthEnd, err := strconv.Atoi(splitEndDate[1])
	if err != nil {
		log.Error("failed to convert string to int", "splitEndDate[1]",
			splitEndDate[1], "error", err.Error())
		tx.Rollback()
		return
	}

	dayEnd, err := strconv.Atoi(splitEndDate[2])
	if err != nil {
		log.Error("failed to convert string to int", "splitEndDate[2]",
			splitEndDate[2], "error", err.Error())
		tx.Rollback()
		return
	}

	start := time.Date(yearStart, time.Month(monthStart), dayStart, startTime.Hour,
		startTime.Min, 0, 0, location)
	end := time.Date(yearEnd, time.Month(monthEnd), dayEnd, endTime.Hour,
		endTime.Min, 0, 0, location)

	//check if start not in past
	if !start.After(time.Now()) {
		v.ErrorKey("validation.calendar.event.exception.start.in.past")
		return
	}

	//ckeck if start before end
	if !start.Before(end) {
		v.ErrorKey("validation.calendar.event.exception.start.after.end")
		return
	}

	//check if exception collides with an existing exception
	var exceptionOverlapping bool
	err = tx.Get(&exceptionOverlapping, stmtExistsOverlappingException, start,
		end, except.CalendarEventID)
	if err != nil {
		log.Error("failed to get exception is overlapping with an existing exception",
			"start", start, "end", except.CalendarEventID, "calendarEventID",
			"error", err.Error())
		tx.Rollback()
		return
	}

	if exceptionOverlapping {
		v.ErrorKey("validation.calendarEvent.overlapping.exception")
		return
	}

	//insert the timestamp into the struct
	except.ExceptionStartDB = start
	except.ExceptionEndDB = end

	return
}

/*Insert an exception. */
func (except *Exception) Insert(tx *sqlx.Tx, v *revel.Validation) (users EMailsData, err error) {

	txWasNil := (tx == nil)
	if txWasNil {
		tx, err = app.Db.Beginx()
		if err != nil {
			log.Error("failed to begin tx", "error", err.Error())
			return
		}
	}

	//check if all values are correct and the selected timespan is free of other exceptions
	if err = except.validate(v, tx); err != nil {
		return
	} else if v.HasErrors() {
		tx.Rollback()
		return
	}

	//insert exception
	err = tx.Get(except, stmtInsertException, except.CalendarEventID,
		except.ExceptionStartDB, except.ExceptionEndDB, except.Annotation)
	if err != nil {
		log.Error("failed to insert Exception", "exception", *except,
			"error", err.Error())
		tx.Rollback()
		return
	}

	//get all slots that are booked during the exception for e-mail data
	err = tx.Select(&users, stmtSelectSlotsDuringException, except.CalendarEventID,
		except.ExceptionStartDB, except.ExceptionEndDB, app.TimeZone)
	if err != nil {
		log.Error("failed to get slots during exception", "exception", *except,
			"error", err.Error())
		tx.Rollback()
		return
	}

	//get more detailed user data
	for idx := range users {
		users[idx].User.ID = users[idx].UserID
		if err = users[idx].User.Get(tx); err != nil {
			return
		}
	}

	//delete slots
	_, err = tx.Exec(stmtDeleteSlotsDuringException, except.CalendarEventID,
		except.ExceptionStartDB, except.ExceptionEndDB)
	if err != nil {
		log.Error("failed to delete slots within exception timespan", "exception", *except,
			"error", err.Error())
		tx.Rollback()
		return
	}

	if txWasNil {
		tx.Commit()
	}
	return
}

/*Update an exception. */
func (except *Exception) Update(v *revel.Validation) (users EMailsData, err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	//delete old exception
	if err = except.Delete(tx); err != nil {
		return
	}
	//insert new exception
	if users, err = except.Insert(tx, v); err != nil {
		return
	}

	tx.Commit()
	return
}

/*Delete an exception. */
func (except *Exception) Delete(tx *sqlx.Tx) (err error) {

	txWasNil := (tx == nil)
	if txWasNil {
		tx, err = app.Db.Beginx()
		if err != nil {
			log.Error("failed to begin tx", "error", err.Error())
			return
		}
	}

	_, err = tx.Exec(stmtDeleteException, except.ID)
	if err != nil {
		log.Error("failed to delete exception", "exception", *except,
			"error", err.Error())
		tx.Rollback()
		return
	}

	if txWasNil {
		tx.Commit()
	}
	return
}

/*Get all current or upcoming exceptions. */
func (excepts *Exceptions) Get(tx *sqlx.Tx, eventID *int) (err error) {

	err = tx.Select(excepts, stmtSelectExceptions, *eventID, app.TimeZone)
	if err != nil {
		log.Error("failed to get exceptions", "eventID", *eventID,
			"error", err.Error())
		tx.Rollback()
	}

	return
}

const (
	stmtSelectExceptionsOfWeek = `
    SELECT id, calendar_event_id, exception_start, exception_end, annotation
    FROM calendar_exceptions
    WHERE
		calendar_event_id = $1
		AND(	($2 >= exception_start AND $2 < exception_end)
			OR 	($3 > exception_start AND $3 <= exception_end)
			OR 	($2 <= exception_start AND $3 >= exception_end )
		)
		ORDER BY exception_start ASC
  `

	stmtInsertException = `
		INSERT INTO calendar_exceptions (
			 calendar_event_id, exception_start, exception_end, annotation
			)
		VALUES (
				$1, $2, $3, $4
		)
		RETURNING id
	`

	stmtExistsOverlappingException = `
		SELECT EXISTS (
			SELECT true
			FROM calendar_exceptions
			WHERE calendar_event_id = $3
				AND (
					($1 >= exception_start AND $1 < exception_end)
					OR ($2 <= exception_end AND $2 > exception_start)
					OR (($1 <= exception_start) AND ($2 >= exception_end))
				)
		) AS exception_overlapping
	`

	stmtSelectSlotsDuringException = `
		SELECT c.title AS course_title, e.title AS event_title, c.id AS course_id,
			TO_CHAR (s.start_time AT TIME ZONE $4, 'YYYY-MM-DD HH24:MI') AS start,
			TO_CHAR (s.end_time AT TIME ZONE $4, 'YYYY-MM-DD HH24:MI') AS end,
			s.user_id AS user_id
		FROM slots s JOIN day_templates d ON d.id = s.day_tmpl_id
			JOIN calendar_events e ON e.id = d.calendar_event_id
			JOIN courses c ON c.id = e.course_id
		WHERE e.id = $1
		AND (
			($2 >= s.start_time AND $2 < s.end_time)
			OR ($3 <= s.end_time AND $3 > s.start_time)
			OR (($2 <= s.start_time) AND ($3 >= s.end_time))
		)
	`

	stmtDeleteSlotsDuringException = `
		DELETE
		FROM slots
		WHERE id IN (
			SELECT s.id
			FROM slots s JOIN day_templates t ON t.id = s.day_tmpl_id
			WHERE t.calendar_event_id = $1
				AND (
					($2 >= s.start_time AND $2 < s.end_time)
					OR ($3 <= s.end_time AND $3 > s.start_time)
					OR	(($2 <= s.start_time) AND ($3 >= s.end_time))
				)
		)
	`

	stmtSelectExceptions = `
    SELECT id, calendar_event_id, annotation,
			TO_CHAR (exception_start AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') AS exception_start_str,
			TO_CHAR (exception_end AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') AS exception_end_str
    FROM calendar_exceptions
    WHERE calendar_event_id = $1
			AND exception_end > now()
		ORDER BY exception_start ASC
  `

	stmtDeleteException = `
	DELETE
	FROM calendar_exceptions
	WHERE id = $1
	`
)
