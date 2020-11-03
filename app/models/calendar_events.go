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

	//Exeptions of this week [0....6]
	ExceptionsAtWeek ExceptionsAtWeek

	ScheduleWeek []Schedule
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
func (event *CalendarEvent) Get(tx *sqlx.Tx, courseID *int, monday time.Time) (err error) {

	txWasNil := (tx == nil)
	if txWasNil {
		tx, err = app.Db.Beginx()
		if err != nil {
			log.Error("failed to begin tx", "error", err.Error())
			return
		}
	}

	err = tx.Get(event, stmtSelectCalendarEvent, *courseID, event.ID)
	if err != nil {
		log.Error("failed to get CalendarEvents of course", "monday", monday, "course ID", *courseID,
			"error", err.Error())
		tx.Rollback()
		return
	}

	//get all day_templates of this event
	err = event.Days.Get(tx, &event.ID, monday)
	if err != nil {
		return
	}

	event.ExceptionsAtWeek = append(event.ExceptionsAtWeek, Exceptions{}, Exceptions{}, Exceptions{},
		Exceptions{}, Exceptions{}, Exceptions{}, Exceptions{})
	//get exceptions
	for i := 0; i < 7; i++ {
		err = event.ExceptionsAtWeek[i].Get(tx, monday, i)
		if err != nil {
			return
		}
	}

	//Tfor ech day get scedule out of all dayTemps of that day and make a
	//		scedule for whole day with exeptions
	for dayIndex := 0; dayIndex < 7; dayIndex++ {
		tmplsOfDay := event.Days[dayIndex]

		daySchedule := Schedule{}
		//generate blocked and free blocks depending on day_templates

		//set blocked slot from 0 to start from first day-template
		if tmplsOfDay[0].StartTime != "00:00" {
			daySchedule = append(daySchedule, ScheduleEntry{"00:00", tmplsOfDay[0].StartTime, BLOCKED})
		}

		/*insert all slots and free space of a dayTemplate and
		the blocked space between the next day template*/
		for i := range tmplsOfDay {

			if i != 0 {
				//check for blocked space to day tmpl upfront
				if tmplsOfDay[i].StartTime != daySchedule[len(daySchedule)-1].EndTime {
					daySchedule = append(daySchedule, ScheduleEntry{daySchedule[len(daySchedule)-1].EndTime,
						tmplsOfDay[i].StartTime, BLOCKED})
				}
			}

			for j := range tmplsOfDay[i].Slots {

				//get convert time's to string
				slotStartTime := Custom_time{"", tmplsOfDay[i].Slots[j].StartTimestamp.Hour(),
					tmplsOfDay[i].Slots[j].StartTimestamp.Minute()}
				slotStartTime.GernerateValueString()

				slotEndTime := Custom_time{"", tmplsOfDay[i].Slots[j].EndTimestamp.Hour(),
					tmplsOfDay[i].Slots[j].EndTimestamp.Minute()}
				slotEndTime.GernerateValueString()

				//check if first slot needs free space upfront
				if j == 0 {

					//insert free ScheduleEntry at start if necessary
					if tmplsOfDay[i].StartTime != slotStartTime.Value {
						daySchedule = append(daySchedule, ScheduleEntry{tmplsOfDay[i].StartTime,
							slotStartTime.Value, EMPTY})
					}
				} else { // check for free space to Schedule entry before current entry
					if daySchedule[len(daySchedule)-1].EndTime != slotStartTime.Value {
						daySchedule = append(daySchedule, ScheduleEntry{daySchedule[len(daySchedule)-1].EndTime,
							slotStartTime.Value, EMPTY})
					}
				}

				//insert slot as ScheduleEntry
				daySchedule = append(daySchedule, ScheduleEntry{slotStartTime.Value, slotEndTime.Value, SLOT})

			} // END of for loop (slots of TmplsOfDay)

			//check for free space from last slot to end of dayTemplate
			if tmplsOfDay[i].EndTime != daySchedule[len(daySchedule)-1].EndTime {
				daySchedule = append(daySchedule, ScheduleEntry{daySchedule[len(daySchedule)-1].EndTime,
					tmplsOfDay[i].EndTime, EMPTY})
			}

		} // END of for loop (TmplsOfDay)

		//check for blocked space from end of last dayTemplate to 24:00
		if daySchedule[len(daySchedule)-1].EndTime != "24:00" {
			daySchedule = append(daySchedule, ScheduleEntry{daySchedule[len(daySchedule)-1].EndTime,
				"24:00", BLOCKED})
		}

		event.ScheduleWeek = append(event.ScheduleWeek, daySchedule)

	}

	//set the current week
	_, event.Week = monday.ISOWeek()

	if txWasNil {
		tx.Commit()
	}

	return
}

//helper functions for Schedule
func insertScheduleEntry(array []ScheduleEntry, element ScheduleEntry, i int) []ScheduleEntry {
	return append(array[:i], append([]ScheduleEntry{element}, array[i:]...)...)
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
	//don't allow the deletion of calendar events if users are enrolled in them
	tx.Get(notEmpty, stmtUsersExist, event.ID)
	if err != nil {
		log.Error("failed to get CalendarEvents of course", "event ID", event.ID,
			"error", err.Error())
		tx.Rollback()
		return
	}

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

	stmtSelectCalendarEvent = `
		SELECT id, course_id, title, annotation
		FROM calendar_events
		WHERE course_id = $1
			AND id = $2
		ORDER BY id ASC
	`

	stmtGetCourseIDByCalendarEvent = `
		SELECT course_id
		FROM calendar_events
		WHERE id = $1
	`

	stmtUsersExist = `
		SELECT EXISTS (
			SELECT true
			FROM day_templates t JOIN slots s ON t.id = s.day_tmpl_id
			WHERE calendar_event_id = $1
		) AS not_empty
	`
)
