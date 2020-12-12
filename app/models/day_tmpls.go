package models

import (
	"time"
	"turm/app"

	"github.com/jmoiron/sqlx"
	"github.com/revel/revel"
)

/*Days of a week. */
type Days []Day

/*Day contains all day templates of a specific week day. */
type Day struct {
	Date     string
	DayTmpls DayTmpls
}

/*DayTmpls holds all day templates of a specific week day. */
type DayTmpls []DayTmpl

/*DayTmpl is a section of a week day (Monday - Sunday). */
type DayTmpl struct {
	ID              int    `db:"id"`
	CalendarEventID int    `db:"calendar_event_id"`
	StartTime       string `db:"start_time"`
	EndTime         string `db:"end_time"`
	Interval        int    `db:"interval"`
	DayOfWeek       int    `db:"day_of_week"` //must be an integer between [0, 6]

	Slots Slots
}

/*Schedule is a helper struct to display a day template at the front end. */
type Schedule struct {
	Date    string
	Entries []ScheduleEntry
	InPast  bool
}

/*ScheduleEntry containing all information to print a section of a day template. */
type ScheduleEntry struct {
	StartTime string
	EndTime   string //should be the same as the subsequent start time
	Interval  int
	Type      ScheduleEntryType

	UserID string
	SlotID int
}

/*Insert a day template. */
func (tmpl *DayTmpl) Insert(tx *sqlx.Tx, v *revel.Validation) (err error) {

	txWasNil := (tx == nil)
	if txWasNil {
		tx, err = app.Db.Beginx()
		if err != nil {
			log.Error("failed to begin tx", "error", err.Error())
			return
		}
	}

	if tmpl.EndTime == "00:00" {
		tmpl.EndTime = "24:00"
	}

	if v != nil {
		if tmpl.validate(v, tx); v.HasErrors() {
			tx.Rollback()
			return
		}
	}

	err = tx.Get(tmpl, stmtInsertDayTemplate, tmpl.CalendarEventID, tmpl.StartTime,
		tmpl.EndTime, tmpl.Interval, tmpl.DayOfWeek)
	if err != nil {
		log.Error("failed to insert day template", "tmpl", *tmpl,
			"error", err.Error())
		tx.Rollback()
		return
	}

	if txWasNil {
		tx.Commit()
	}
	return
}

/*Update a day tmpl. */
func (tmpl *DayTmpl) Update(v *revel.Validation) (users []EMailData, err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	if tmpl.EndTime == "00:00" {
		tmpl.EndTime = "24:00"
	}

	//validate new DayTemplate
	tmpl.validate(v, tx)
	if v.HasErrors() {
		tx.Rollback()
		return
	}

	var slots Slots
	if err = slots.GetAll(tx, tmpl.ID); err != nil {
		return
	}

	slotStartTime := CustomTime{}
	slotEndTime := CustomTime{}

	tmplStartTime := CustomTime{}
	tmplStartTime.SetTime(tmpl.StartTime)

	tmplEndTime := CustomTime{}
	tmplEndTime.SetTime(tmpl.EndTime)

	for _, slot := range slots {

		slotStartTime.Hour, slotStartTime.Min, _ = slot.Start.Clock()
		slotEndTime.Hour, slotEndTime.Min, _ = slot.End.Clock()

		//all the cases in which a slot has to be deleted
		hasToBeDeleted := slotEndTime.After(&tmplEndTime) ||
			tmplStartTime.After(&slotStartTime) || //delete if slot collides with new times
			int(slot.Start.Weekday())-1 != tmpl.DayOfWeek || //delete if weekday is changed
			(slotEndTime.Sub(&tmplStartTime)%tmpl.Interval) != 0 || //checks for interval steps
			(slotStartTime.Sub(&tmplStartTime)%tmpl.Interval) != 0

		//check if slot is not within the new time
		if hasToBeDeleted {

			//don't allow update if a slot in this template is currently running
			if slot.Start.Before(time.Now()) && slot.End.After(time.Now()) {
				v.ErrorKey("validation.calendar.event.slot.running")
				tx.Rollback()
				return
			}

			//append e-mail data (if slot is upcoming)
			if slot.End.After(time.Now()) {

				//get e-mail data
				data := EMailData{}
				data.User.ID = slot.UserID
				if err = data.User.Get(tx); err != nil {
					return
				}

				err = tx.Get(&data, stmtGetSlotEMailData, slot.ID, app.TimeZone)
				if err != nil {
					log.Error("failed to get slot data for e-mail", "slotID", slot.ID,
						"error", err.Error())
					tx.Rollback()
					return
				}

				users = append(users, data)
			}

			//delete slot
			if err = deleteByID("id", "slots", slot.ID, tx); err != nil {
				return
			}
		}
	}

	//update day template
	if err = updateByID(tx, "start_time", "day_templates", tmpl.StartTime, tmpl.ID, tmpl); err != nil {
		return
	}
	if err = updateByID(tx, "end_time", "day_templates", tmpl.EndTime, tmpl.ID, tmpl); err != nil {
		return
	}
	if err = updateByID(tx, "interval", "day_templates", tmpl.Interval, tmpl.ID, tmpl); err != nil {
		return
	}
	if err = updateByID(tx, "day_of_week", "day_templates", tmpl.DayOfWeek, tmpl.ID, tmpl); err != nil {
		return
	}

	tx.Commit()
	return
}

/*Duplicate all day_templates if the eventIDOld into the eventIDNew*/
func (tmpls *DayTmpls) Duplicate(tx *sqlx.Tx, eventIDNew, eventIDOld *int) (err error) {

	_, err = tx.Exec(stmtDuplicateDayTemplates, *eventIDNew, *eventIDOld)
	if err != nil {
		log.Error("failed to duplicate day templates", "eventIDNew",
			*eventIDNew, "eventIDOld", *eventIDOld, "error", err.Error())
		tx.Rollback()
	}

	return
}

/*Delete a day template if it has no slots. */
func (tmpl *DayTmpl) Delete() (users EMailsData, err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	//get all slots of this day template
	var slots Slots
	if err = slots.GetAll(tx, tmpl.ID); err != nil {
		return
	}

	//get all users that have booked slots for this day template (in the future)
	for _, slot := range slots {

		//append e-mail data (if slot is upcoming)
		if slot.End.After(time.Now()) {

			//get e-mail data
			data := EMailData{}
			data.User.ID = slot.UserID
			if err = data.User.Get(tx); err != nil {
				return
			}

			err = tx.Get(&data, stmtGetSlotEMailData, slot.ID, app.TimeZone)
			if err != nil {
				log.Error("failed to get slot data for e-mail", "slotID", slot.ID,
					"error", err.Error())
				tx.Rollback()
				return
			}

			users = append(users, data)
		}
	}

	//delete day template
	if err = deleteByID("id", "day_templates", tmpl.ID, tx); err != nil {
		return
	}

	tx.Commit()
	return
}

/*Get all days of a calendar event for a specific week. */
func (days *Days) Get(tx *sqlx.Tx, calendarEventID *int, monday time.Time,
	participants bool) (err error) {

	txWasNil := (tx == nil)
	if txWasNil {
		tx, err = app.Db.Beginx()
		if err != nil {
			log.Error("failed to begin tx", "error", err.Error())
			return
		}
	}

	weekDay := monday

	//init a slice for each week day
	*days = append(*days, Day{}, Day{}, Day{}, Day{}, Day{}, Day{}, Day{})

	//iterate week days
	for i := 0; i < 7; i++ {

		//set the date
		(*days)[i].Date = weekDay.Format("02.01.")
		weekDay = weekDay.AddDate(0, 0, 1)

		//get templates of each day
		err = tx.Select(&(*days)[i].DayTmpls, stmtSelectDayTmpls, *calendarEventID, i)
		if err != nil {
			log.Error("failed to get day tmpls by week day", "calendarEventID",
				*calendarEventID, "i", i, "error", err.Error())
			tx.Rollback()
			return
		}

		for j := range (*days)[i].DayTmpls {

			//get slots
			err = ((*days)[i].DayTmpls)[j].Slots.Get(tx, ((*days)[i].DayTmpls)[j].ID, monday, i)
			if err != nil {
				return
			}

			//format time
			if ((*days)[i].DayTmpls)[j].EndTime == "00:00" {
				((*days)[i].DayTmpls)[j].EndTime = "24:00"
			}

			//TODO: get if user is allowed to view matriculation numbers

			if participants {

				//get detailed user information
				for idx, slot := range ((*days)[i].DayTmpls)[j].Slots {
					((*days)[i].DayTmpls)[j].Slots[idx].User.ID = slot.UserID
					err = ((*days)[i].DayTmpls)[j].Slots[idx].User.Get(tx)
					if err != nil {
						return
					}
				}
			}

		}
	}

	if txWasNil {
		tx.Commit()
	}
	return
}

//validate a day template
func (tmpl *DayTmpl) validate(v *revel.Validation, tx *sqlx.Tx) {

	//check for valid times
	start := CustomTime{}
	isValidTime1 := start.SetTime(tmpl.StartTime)

	end := CustomTime{}
	isValidTime2 := end.SetTime(tmpl.EndTime)

	if !isValidTime1 || !isValidTime2 {
		v.ErrorKey("validation.invalid.timestamp")
	}

	if !end.After(&start) {
		v.ErrorKey("validation.calendar.event.start.after.end")
	}

	//check step distance
	distance := float64(start.Sub(&end))
	multiplier := distance / float64(tmpl.Interval)
	if multiplier-float64(int(multiplier)) != 0.0 {
		v.ErrorKey("validation.calendar.event.wrong.interval")
	}

	if !v.HasErrors() {

		//check if template collides with other templates on that day
		var overlaps bool

		err := tx.Get(&overlaps, stmtGetOverlappingTmpls, start.Value,
			end.Value, tmpl.DayOfWeek, tmpl.CalendarEventID, tmpl.ID)
		if err != nil {
			log.Error("failed to validate if day templates overlap each other", "day tmpl",
				*tmpl, "error", err.Error())
			v.ErrorKey("error.db")
			tx.Rollback()
			return
		}

		if overlaps {
			v.ErrorKey("validation.calendar.event.tmpls.overlap")
		}
	}
}

const (
	stmtInsertDayTemplate = `
    INSERT INTO day_templates (
        calendar_event_id, start_time, end_time, interval, day_of_week
			)
    VALUES ($1, $2, $3, $4, $5)
    RETURNING id
  `

	stmtSelectDayTemplatesFromWeekDay = `
    SELECT id, start_time, end_time, interval
    FROM day_templates
    WHERE day_of_week = $1
			AND calendar_event_id = $2
		ORDER BY start_time ASC
  `

	stmtSelectDayTmpls = `
    SELECT id, calendar_event_id,
			TO_CHAR (date '2001-09-28' + start_time, 'HH24:MI') AS start_time,
			TO_CHAR (date '2001-09-28' + end_time, 'HH24:MI') AS end_time,
			interval, day_of_week
    FROM day_templates
    WHERE calendar_event_id = $1
      AND day_of_week = $2
    ORDER BY start_time ASC
  `

	stmtDuplicateDayTemplates = `
		INSERT INTO day_templates
			(calendar_event_id, start_time, end_time, interval,
				day_of_week)
		(
			SELECT
				$1 AS calendar_event_id, start_time, end_time, interval,
				 day_of_week
			FROM day_templates
			WHERE calendar_event_id = $2
		)
	`

	stmtGetOverlappingTmpls = `
		SELECT EXISTS(
			SELECT true
			FROM day_templates
			WHERE $4 = calendar_event_id
				AND day_of_week = $3
				AND id != $5
				AND (
					($1 >= start_time AND $1 < end_time)
					OR 	($2 <= end_time AND $2 > start_time)
					OR 	(($1 <= start_time) AND ($2 >= end_time))
				)
		) AS overlaps
	`
)
