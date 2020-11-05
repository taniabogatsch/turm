package models

import (
	"database/sql"
	"time"
	"turm/app"

	"github.com/jmoiron/sqlx"
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

	//exeptions of this week [0....6]
	ExceptionsOfWeek ExceptionsOfWeek
	//transformed schedule for easy front end usage
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

/*Get all CalendarEvents by the provided monday. */
func (events *CalendarEvents) Get(tx *sqlx.Tx, courseID *int, monday time.Time) (err error) {

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
		log.Error("failed to get calendar events of course", "course ID", *courseID,
			"error", err.Error())
		tx.Rollback()
		return
	}

	for i := range *events {
		//get all day templates of this event
		err = (*events)[i].Days.Get(tx, &(*events)[i].ID, monday)
		if err != nil {
			return
		}

		//get the slot schedule
		if err = (*events)[i].getSchedule(tx, monday); err != nil {
			return
		}

		//set the current week
		_, (*events)[i].Week = monday.ISOWeek()
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

	//get general event information
	err = tx.Get(event, stmtGetCalendarEvent, *courseID, event.ID)
	if err != nil {
		log.Error("failed to get calendar event of course", "monday", monday, "course ID", *courseID,
			"error", err.Error())
		tx.Rollback()
		return
	}

	//get all day templates of this event
	if err = event.Days.Get(tx, &event.ID, monday); err != nil {
		return
	}

	//get the slot schedule
	if err = event.getSchedule(tx, monday); err != nil {
		return
	}

	//set the current week
	_, event.Week = monday.ISOWeek()

	if txWasNil {
		tx.Commit()
	}
	return
}

func (event *CalendarEvent) getSchedule(tx *sqlx.Tx, monday time.Time) (err error) {

	//get the exceptions of each day of that week (as defined by monday)
	event.ExceptionsOfWeek = append(event.ExceptionsOfWeek, Exceptions{}, Exceptions{},
		Exceptions{}, Exceptions{}, Exceptions{}, Exceptions{}, Exceptions{})
	for i := 0; i < 7; i++ {
		if err = event.ExceptionsOfWeek[i].Get(tx, monday, i); err != nil {
			return
		}
	}

	//prepare a schedule for the whole week by looping all day templates and
	//their slots for each day respectively
	for _, tmplsOfDay := range event.Days {

		//generate blocked and free blocks of this schedule
		schedule := Schedule{Date: "28.09."} //TODO: set the date

		if len(tmplsOfDay) != 0 {

			//set blocked slot from 0 to start of the first day template
			if tmplsOfDay[0].StartTime != "00:00" {
				schedule.Entries = append(schedule.Entries,
					ScheduleEntry{"00:00", tmplsOfDay[0].StartTime, BLOCKED})
			}

			//insert all slots and free spaces of a day template and
			//the blocked space between this day template and the next day template
			for i := range tmplsOfDay {

				//if two day templates are not exactly subsequent to each other,
				//then insert a BLOCKED schedule entry
				if i != 0 {
					if tmplsOfDay[i].StartTime != schedule.Entries[len(schedule.Entries)-1].EndTime {
						schedule.Entries = append(schedule.Entries,
							ScheduleEntry{
								schedule.Entries[len(schedule.Entries)-1].EndTime,
								tmplsOfDay[i].StartTime,
								BLOCKED},
						)
					}
				}

				//insert all BOOKED and FREE schedule entries for the current day template
				for j := range tmplsOfDay[i].Slots {

					//get start time as string
					slotStart := CustomTime{"", tmplsOfDay[i].Slots[j].Start.Hour(),
						tmplsOfDay[i].Slots[j].Start.Minute()}
					slotStart.String()

					//get end time as string
					slotEnd := CustomTime{"", tmplsOfDay[i].Slots[j].End.Hour(),
						tmplsOfDay[i].Slots[j].End.Minute()}
					slotEnd.String()

					//check if there is free space before the first BOOKED slot
					if j == 0 {
						//insert FREE schedule entry
						if tmplsOfDay[i].StartTime != slotStart.Value {
							schedule.Entries = append(schedule.Entries, ScheduleEntry{tmplsOfDay[i].StartTime,
								slotStart.Value, FREE})
						}
					} else {
						//check for FREE space between two slots
						if schedule.Entries[len(schedule.Entries)-1].EndTime != slotStart.Value {
							schedule.Entries = append(schedule.Entries, ScheduleEntry{schedule.Entries[len(schedule.Entries)-1].EndTime,
								slotStart.Value, FREE})
						}
					}

					//insert slot as schedule entry
					schedule.Entries = append(schedule.Entries, ScheduleEntry{slotStart.Value, slotEnd.Value,
						SLOT})

				} //end of for loop of slots

				//check for FREE space from the last slot to the end of the day template
				if tmplsOfDay[i].EndTime != schedule.Entries[len(schedule.Entries)-1].EndTime {
					schedule.Entries = append(schedule.Entries, ScheduleEntry{schedule.Entries[len(schedule.Entries)-1].EndTime,
						tmplsOfDay[i].EndTime, FREE})
				}

			} //end of for loop of day templates

			//check for BLOCKED space from the end of the last day template to 24:00
			if schedule.Entries[len(schedule.Entries)-1].EndTime != "24:00" {
				schedule.Entries = append(schedule.Entries, ScheduleEntry{schedule.Entries[len(schedule.Entries)-1].EndTime,
					"24:00", BLOCKED})
			}

		} else {
			//no day templates for this day
			schedule.Entries = append(schedule.Entries,
				ScheduleEntry{"00:00", "24:00", BLOCKED})
		}

		event.ScheduleWeek = append(event.ScheduleWeek, schedule)
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
func (event *CalendarEvent) Delete() (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	//TODO: get all users that have booked slots for this event (in the future)
	//TODO: return these and write them an e-mail

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
		VALUES ($1, $2)
		RETURNING id
	`

	stmtSelectCalendarEvents = `
		SELECT id, course_id, title, annotation
		FROM calendar_events
		WHERE course_id = $1
		ORDER BY id ASC
	`

	stmtGetCalendarEvent = `
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
)
