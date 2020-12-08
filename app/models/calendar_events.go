package models

import (
	"database/sql"
	"math"
	"strconv"
	"strings"
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
	Monday time.Time
	Week   int
	Year   int

	//day templates for this week [0...6]
	Days DayTmpls

	//all upcoming exceptions
	Exceptions Exceptions

	//exeptions of this week
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

		//get the exceptions of each day of that week (as defined by monday)
		if err = (*events)[i].ExceptionsOfWeek.Get(tx, &(*events)[i].ID, monday); err != nil {
			return
		}

		//set the current week
		(*events)[i].Monday = monday
		_, (*events)[i].Week = monday.ISOWeek()
		(*events)[i].Year = monday.Year()

		//get the slot schedule
		if err = (*events)[i].getSchedule(tx, monday); err != nil {
			return
		}

		//get all calendar exceptions
		if err = (*events)[i].Exceptions.Get(tx, &(*events)[i].ID); err != nil {
			return
		}
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

	//get the exceptions of each day of that week (as defined by monday)
	if err = event.ExceptionsOfWeek.Get(tx, &event.ID, monday); err != nil {
		return
	}

	//set the current week
	event.Monday = monday
	_, event.Week = monday.ISOWeek()
	event.Year = monday.Year()

	//get the slot schedule
	if err = event.getSchedule(tx, monday); err != nil {
		return
	}

	if txWasNil {
		tx.Commit()
	}
	return
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

func (event *CalendarEvent) getSchedule(tx *sqlx.Tx, monday time.Time) (err error) {

	day := monday

	//prepare a schedule for the whole week by looping all day templates and
	//their slots for each day respectively
	for _, tmplsOfDay := range event.Days {

		inPast := day.Before(time.Now())

		//generate blocked and free blocks of this schedule
		schedule := Schedule{Date: day.Format("02.01."), InPast: inPast}
		day = day.AddDate(0, 0, 1)

		if len(tmplsOfDay) != 0 {

			//set blocked slot from 0 to start of the first day template
			if tmplsOfDay[0].StartTime != "00:00" {
				schedule.Entries = append(schedule.Entries,
					ScheduleEntry{"00:00", tmplsOfDay[0].StartTime, 0, BLOCKED, "0", 0})
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
								0, BLOCKED, "0", 0},
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
								slotStart.Value, tmplsOfDay[i].Interval, FREE, "0", 0})
						}
					} else {
						//check for FREE space between two slots
						if schedule.Entries[len(schedule.Entries)-1].EndTime != slotStart.Value {
							schedule.Entries = append(schedule.Entries, ScheduleEntry{schedule.Entries[len(schedule.Entries)-1].EndTime,
								slotStart.Value, tmplsOfDay[i].Interval, FREE, "0", 0})
						}
					}

					//insert slot as schedule entry
					schedule.Entries = append(schedule.Entries, ScheduleEntry{slotStart.Value, slotEnd.Value,
						tmplsOfDay[i].Interval, SLOT,
						strconv.Itoa(tmplsOfDay[i].Slots[j].UserID),
						tmplsOfDay[i].Slots[j].ID})

				} //end of for loop of slots

				if len(schedule.Entries) > 0 {
					//check for FREE space from the last slot to the end of the day template
					if tmplsOfDay[i].EndTime != schedule.Entries[len(schedule.Entries)-1].EndTime {
						schedule.Entries = append(schedule.Entries, ScheduleEntry{schedule.Entries[len(schedule.Entries)-1].EndTime,
							tmplsOfDay[i].EndTime, tmplsOfDay[i].Interval, FREE, "0", 0})
					}
				} else {
					schedule.Entries = append(schedule.Entries, ScheduleEntry{tmplsOfDay[i].StartTime,
						tmplsOfDay[i].EndTime, tmplsOfDay[i].Interval, FREE, "0", 0})
				}

			} //end of for loop of day templates

			//check for BLOCKED space from the end of the last day template to 24:00
			if schedule.Entries[len(schedule.Entries)-1].EndTime != "24:00" {
				schedule.Entries = append(schedule.Entries, ScheduleEntry{schedule.Entries[len(schedule.Entries)-1].EndTime,
					"24:00", 0, BLOCKED, "0", 0})
			}

		} else {
			//no day templates for this day
			schedule.Entries = append(schedule.Entries,
				ScheduleEntry{"00:00", "24:00", 0, BLOCKED, "0", 0})
		}

		//after each day, loop all exceptions of the week and
		//test if any overlap with that day

		for eIdx := range event.ExceptionsOfWeek {

			//used to remember the first and last slot overlapping the exception
			startSlotIdx := -1
			var startEntry ScheduleEntry
			endSlotIdx := -1
			var endEntry ScheduleEntry

			//loop entries to find those overlapping with the exception
			for idx := range schedule.Entries {

				//get start and end
				start, err := parseDate(tx, strconv.Itoa(event.Year), schedule.Date,
					schedule.Entries[idx].StartTime)
				if err != nil {
					return err
				}
				end, err := parseDate(tx, strconv.Itoa(event.Year), schedule.Date,
					schedule.Entries[idx].EndTime)
				if err != nil {
					return err
				}

				if schedule.Entries[idx].EndTime == "24:00" {
					end = end.AddDate(0, 0, 1)
				}

				//test if this entry overlaps with the exception and is thus the first
				//overlapping entry
				if startSlotIdx == -1 && event.ExceptionsOfWeek[eIdx].ExceptionStartDB.Before(end) &&
					event.ExceptionsOfWeek[eIdx].ExceptionEndDB.After(start) {
					startSlotIdx = idx
					startEntry = schedule.Entries[idx]
				}

				//test if any of the schedule entries is the last one overlapping with this exception
				if end.After(event.ExceptionsOfWeek[eIdx].ExceptionEndDB) ||
					end.Equal(event.ExceptionsOfWeek[eIdx].ExceptionEndDB) {
					endSlotIdx = idx
					endEntry = schedule.Entries[idx]
					break
				}
			}

			//set to last index if the exception did not end on that day
			if endSlotIdx == -1 {
				endSlotIdx = len(schedule.Entries) - 1
				endEntry = schedule.Entries[len(schedule.Entries)-1]
			}

			//if an overlapping exception was found
			if startSlotIdx != -1 {

				//remove all slots overlapping with the exception [startSlotIdx, endSlotIdx]
				if endSlotIdx == len(schedule.Entries)-1 {
					schedule.Entries = schedule.Entries[:startSlotIdx]
				} else {
					schedule.Entries = append(schedule.Entries[:startSlotIdx],
						schedule.Entries[endSlotIdx+1:]...)
				}

				//convert start and end time of schedule entry to date + time
				start, err := parseDate(tx, strconv.Itoa(event.Year), schedule.Date,
					startEntry.StartTime)
				if err != nil {
					return err
				}

				end, err := parseDate(tx, strconv.Itoa(event.Year), schedule.Date,
					endEntry.StartTime)
				if err != nil {
					return err
				}

				//get the starting time of the exception with respect to the step interval
				startTime := getExceptionScheduleTimes(startEntry.Interval,
					start, event.ExceptionsOfWeek[eIdx].ExceptionStartDB, true)

				endTime := getExceptionScheduleTimes(endEntry.Interval,
					end, event.ExceptionsOfWeek[eIdx].ExceptionEndDB, false)

				//test if the exception starts and ends at the same day as the current schedule
				if event.ExceptionsOfWeek[eIdx].ExceptionStartDB.YearDay() == start.YearDay() &&
					event.ExceptionsOfWeek[eIdx].ExceptionStartDB.Year() == start.Year() {

					if event.ExceptionsOfWeek[eIdx].ExceptionEndDB.YearDay() == end.YearDay() &&
						event.ExceptionsOfWeek[eIdx].ExceptionEndDB.Year() == end.Year() {

						//insert the entry slice upfront the exception, FREE or BLOCKED
						if startEntry.StartTime != startTime {
							if startEntry.Interval != 0 {
								schedule.Entries = insertScheduleEntry(schedule.Entries,
									ScheduleEntry{startEntry.StartTime, startTime,
										startEntry.Interval, FREE, "0", 0}, startSlotIdx)
								startSlotIdx++
							} else {
								schedule.Entries = insertScheduleEntry(schedule.Entries,
									ScheduleEntry{startEntry.StartTime, startTime,
										startEntry.Interval, BLOCKED, "0", 0}, startSlotIdx)
								startSlotIdx++
							}
						}

						//insert the EXCEPTION entry
						schedule.Entries = insertScheduleEntry(schedule.Entries,
							ScheduleEntry{startTime, endTime,
								0, EXCEPTION, "0", 0}, startSlotIdx)
						startSlotIdx++

						//insert the entry slice after the exception, FREE or BLOCKED
						if endTime != endEntry.EndTime {
							if endEntry.Interval != 0 {
								schedule.Entries = insertScheduleEntry(schedule.Entries,
									ScheduleEntry{endTime, endEntry.EndTime,
										endEntry.Interval, FREE, "0", 0}, startSlotIdx)
							} else {
								schedule.Entries = insertScheduleEntry(schedule.Entries,
									ScheduleEntry{endTime, endEntry.EndTime,
										endEntry.Interval, BLOCKED, "0", 0}, startSlotIdx)
							}
						}
					} else { //end is 24:00

						//insert the entry slice upfront the exception, FREE or BLOCKED
						if startEntry.StartTime != startTime {
							if startEntry.Interval != 0 {
								schedule.Entries = insertScheduleEntry(schedule.Entries,
									ScheduleEntry{startEntry.StartTime, startTime,
										startEntry.Interval, FREE, "0", 0}, startSlotIdx)
								startSlotIdx++
							} else {
								schedule.Entries = insertScheduleEntry(schedule.Entries,
									ScheduleEntry{startEntry.StartTime, startTime,
										startEntry.Interval, BLOCKED, "0", 0}, startSlotIdx)
								startSlotIdx++
							}
						}

						schedule.Entries = insertScheduleEntry(schedule.Entries,
							ScheduleEntry{startTime, "24: 00",
								startEntry.Interval, EXCEPTION, "0", 0}, startSlotIdx)

					}
				} else { //exception start at 00:00

					//exception start at 00:00 and ends at 24:00
					if event.ExceptionsOfWeek[eIdx].ExceptionEndDB.YearDay() != end.YearDay() ||
						event.ExceptionsOfWeek[eIdx].ExceptionEndDB.Year() != end.Year() {

						schedule.Entries = insertScheduleEntry(schedule.Entries,
							ScheduleEntry{"00:00", "24:00",
								startEntry.Interval, EXCEPTION, "0", 0}, startSlotIdx)

					} else { //exception only starts at 00:00
						endTime := getExceptionScheduleTimes(endEntry.Interval,
							end, event.ExceptionsOfWeek[eIdx].ExceptionEndDB, false)

						schedule.Entries = insertScheduleEntry(schedule.Entries,
							ScheduleEntry{"00:00", endTime,
								startEntry.Interval, EXCEPTION, "0", 0}, startSlotIdx)

						//insert the entry slice after the exception, FREE or BLOCKED
						if endTime != endEntry.EndTime {
							if endEntry.Interval != 0 {
								schedule.Entries = insertScheduleEntry(schedule.Entries,
									ScheduleEntry{endTime, endEntry.EndTime,
										endEntry.Interval, FREE, "0", 0}, startSlotIdx)
							} else {
								schedule.Entries = insertScheduleEntry(schedule.Entries,
									ScheduleEntry{endTime, endEntry.EndTime,
										endEntry.Interval, BLOCKED, "0", 0}, startSlotIdx+1)
							}
						}
					}
				}
			}
		}
		//insert the schedule of the day in the Week-Schedule slice
		event.ScheduleWeek = append(event.ScheduleWeek, schedule)
	}

	return
}

func parseDate(tx *sqlx.Tx, year, date, str string) (t time.Time, err error) {

	//create start/end date + time from entry to compare with exception start/end date + time
	split := strings.Split(date, ".")
	loc, err := time.LoadLocation(app.TimeZone)
	if err != nil {
		log.Error("failed to parse location", "loc", app.TimeZone,
			"error", err.Error())
		tx.Rollback()
		return
	}

	addDay := false
	value := ""

	//get for start time of entry
	if str == "24:00" { //ParseInLocation is not compatible with 24:00
		value = year + "-" + split[1] + "-" + split[0] + "T" + "00:00:00"
		addDay = true
	} else {
		value = year + "-" + split[1] + "-" + split[0] + "T" + str + ":00"
	}

	t, err = time.ParseInLocation("2006-01-02T15:04:05", value, loc)
	if err != nil {
		log.Error("failed to parse string to time", "value", value, "loc", loc,
			"error", err.Error())
		tx.Rollback()
		return
	}

	if addDay { //if it was 24:00
		t.AddDate(0, 0, 1) //now 00:00 of next day
	}

	return
}

//helper functions for Schedule
func insertScheduleEntry(array []ScheduleEntry, element ScheduleEntry, i int) []ScheduleEntry {
	return append(array[:i], append([]ScheduleEntry{element}, array[i:]...)...)
}

//exceptStart true -> get next Time before , false -> get next time after
func getExceptionScheduleTimes(interval int, sStart time.Time, exceptTime time.Time,
	exceptStart bool) (sTime string) {

	if interval == 0 {
		//TODO: this looks ugly
		return prettyTime(exceptTime.Hour()) + ":" + prettyTime(exceptTime.Minute())
	}

	if exceptTime.After(sStart) {

		rangeMin := (exceptTime.Hour()*60 + exceptTime.Minute()) - (sStart.Hour()*60 + sStart.Minute())
		var min int

		if exceptStart {
			flo := float64(rangeMin) / float64(interval)
			min = int(math.Floor(flo)) * interval
		} else {
			flo := float64(rangeMin) / float64(interval)
			min = int(math.Ceil(flo)) * interval
		}

		minStr, _ := time.ParseDuration(strconv.Itoa(min) + "m")

		sStart = sStart.Add(minStr)

		sHour := prettyTime(sStart.Hour())
		sMin := prettyTime(sStart.Minute())
		sTime = sHour + ":" + sMin

	} else {
		rangeMin := (sStart.Hour()*60 + sStart.Minute()) - (exceptTime.Hour()*60 + exceptTime.Minute())
		var min int

		if exceptStart {
			flo := float64(rangeMin) / float64(interval)
			min = int(math.Floor(flo)) * interval
		} else {
			flo := float64(rangeMin) / float64(interval)
			min = int(math.Ceil(flo)) * interval
		}

		minStr, _ := time.ParseDuration(strconv.Itoa(min) + "m")

		sStart = sStart.Add(minStr)

		sHour := prettyTime(sStart.Hour())
		sMin := prettyTime(sStart.Minute())
		sTime = sHour + ":" + sMin
	}
	return
}

func prettyTime(i int) string {

	if i < 10 {
		return "0" + strconv.Itoa(i)
	}
	return strconv.Itoa(i)
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
		SELECT course_id AS id
		FROM calendar_events
		WHERE id = $1
	`
)
