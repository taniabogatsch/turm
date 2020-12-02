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

/*Get all exceptions of a day. Monday specifies the week for which all exceptions
must be loaded. */
func (excepts *ExceptionsOfWeek) Get(tx *sqlx.Tx, monday time.Time) (err error) {

	endTime := monday.AddDate(0, 0, 7)

	err = tx.Select(excepts, stmtSelectExceptionsOfWeek, monday, endTime)
	if err != nil {
		log.Error("failed to get exceptions of day template", "monday", monday,
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
func (except *Exception) Insert(v *revel.Validation) (data EMailData, users Users,
	err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
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

	//delete all enrolled users in the time span
	var userIDs []int

	//get all IDs of users that booked slots in the timespan of the exception
	err = tx.Select(&userIDs, stmtSelectUserIDsInExeptionTime, except.CalendarEventID,
		except.ExceptionStartDB, except.ExceptionEndDB)
	if err != nil {
		log.Error("failed to get user IDs within exception timespan", "exception", *except,
			"error", err.Error())
		tx.Rollback()
		return
	}

	for _, userID := range userIDs {

		user := User{ID: userID}
		if err = user.Get(tx); err != nil {
			return
		}
		users = append(users, user)
	}

	//get CourseID, CourseTitle and EventTitle
	err = tx.Get(&data, stmtGetCourseInfoByCalendarEvent, except.CalendarEventID)
	if err != nil {
		log.Error("failed to get CourseID, CourseTitle and EventTitle", "exception", *except,
			"error", err.Error())
		tx.Rollback()
		return
	}

	//delete slots
	_, err = tx.Exec(stmtDeleteSlotsInExeptionTime, except.CalendarEventID,
		except.ExceptionStartDB, except.ExceptionEndDB)
	if err != nil {
		log.Error("failed to delete slots within exception timespan", "exception", *except,
			"error", err.Error())
		tx.Rollback()
		return
	}

	tx.Commit()
	return
}

/*Update an exception. */
func (except *Exception) Update(v *revel.Validation) (data EMailData, users Users, err error) {

	//in transaction:
	//TODO: delete old exception
	//TODO: insert new exception
	return
}

/*Delete an exception. */
func (except *Exception) Delete(v *revel.Validation) (err error) {
	//TODO
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
    WHERE (
			($1 >= exception_start AND $1 < exception_end)
			OR ($2 > exception_start AND $2 <= exception_end)
			OR ($1 < exception_start AND $2 > exception_end )
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

	stmtSelectUserIDsInExeptionTime = `
		SELECT s.user_id
		FROM slots s JOIN day_templates t ON t.id = s.day_tmpl_id
		WHERE t.calendar_event_id = $1
		AND (
			($2 >= s.start_time AND $2 < s.end_time)
			OR ($3 <= s.end_time AND $3 > s.start_time)
			OR (($2 <= s.start_time) AND ($3 >= s.end_time))
		)
	`

	stmtGetCourseInfoByCalendarEvent = `
		SELECT c.id AS course_id, c.title AS course_title, ce.title AS event_title
		FROM calendar_events ce JOIN courses c ON ce.course_id = c.id
		WHERE ce.id = $1
	`

	stmtDeleteSlotsInExeptionTime = `
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
)
