package models

import (
	"time"
	"turm/app"

	"github.com/jmoiron/sqlx"
	"github.com/revel/revel"
)

/*DayTmpls of a week for each day. */
type DayTmpls []TmplsOfDay

/*TmplsOfDay contains all day templates of a specific week day. */
type TmplsOfDay []DayTmpl

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
}

/*ScheduleEntry containing all information to print a section of a day template. */
type ScheduleEntry struct {
	StartTime string
	EndTime   string //should be the same as the subsequent start time
	Interval  int
	Type      ScheduleEntryType
}

/*Insert a day template. */
func (tmpl *DayTmpl) Insert(v *revel.Validation) (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	if tmpl.EndTime == "00:00" {
		tmpl.EndTime = "24:00"
	}

	if tmpl.validate(v, tx); v.HasErrors() {
		tx.Rollback()
		return
	}

	err = tx.Get(tmpl, stmtInsertDayTemplate, tmpl.CalendarEventID, tmpl.StartTime,
		tmpl.EndTime, tmpl.Interval, tmpl.DayOfWeek)
	if err != nil {
		log.Error("failed to insert day template", "tmpl", *tmpl,
			"error", err.Error())
		tx.Rollback()
		return
	}

	tx.Commit()
	return
}

/*Update a day tmpl. */
func (tmpl *DayTmpl) Update(v *revel.Validation) (data EMailData, users Users, err error) {

	//map to remember which user is already getting an E-Mail
	userIDs := make(map[int]bool)

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
	slots.GetAll(tx, tmpl.ID)

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
			tmplStartTime.After(&slotStartTime) || //delete if slot colides with new times
			int(slot.Start.Weekday())-1 != tmpl.DayOfWeek || //if weekday is changed delete slot
			(slotEndTime.Sub(&tmplStartTime)%tmpl.Interval) != 0 || //checks for intervall stepps
			(slotStartTime.Sub(&tmplStartTime)%tmpl.Interval) != 0

		//check if slot is not within the new time
		if hasToBeDeleted {

			//dont allow update if a slot in this template is currently running
			if slot.Start.Before(time.Now()) && slot.End.After(time.Now()) {
				v.ErrorKey("message") //TODO: language File
				return
			}

			//slot has to be deleted
			//delete slot
			if err = deleteByID("id", "slots", slot.ID, tx); err != nil {
				return
			}

			//create user for email if slot isnt in past and not in mailing(users) list already
			if slot.End.After(time.Now()) && !userIDs[slot.UserID] {
				user := User{ID: slot.UserID}
				if err = user.Get(tx); err != nil {
					return
				}

				userIDs[slot.UserID] = true
				users = append(users, user)
			}

		}

		//get email data
		err = tx.Get(&data, stmtGetCourseInfoByCalendarEvent, tmpl.CalendarEventID)
		if err != nil {
			log.Error("failed to get CourseID, CourseTitle and EventTitle", "day_template", *tmpl,
				"error", err.Error())
			tx.Rollback()
			return
		}

	}

	//update times
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

/*Delete a day template if it has no slots. */
func (tmpl *DayTmpl) Delete() (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	//TODO: get all users that have booked slots for this day template (in the future)
	//TODO: return these and write them an e-mail

	//delete day template
	if err = deleteByID("id", "day_templates", tmpl.ID, tx); err != nil {
		return
	}

	tx.Commit()
	return
}

/*Get all day templates of a CalendarEvent for a specific week. */
func (dayTmpls *DayTmpls) Get(tx *sqlx.Tx, calendarEventID *int, monday time.Time) (err error) {

	//init a slice for each week day
	*dayTmpls = append(*dayTmpls, TmplsOfDay{}, TmplsOfDay{}, TmplsOfDay{},
		TmplsOfDay{}, TmplsOfDay{}, TmplsOfDay{}, TmplsOfDay{})

	//iterate week days
	for i := 0; i < 7; i++ {

		//get templates of each day
		err = tx.Select(&(*dayTmpls)[i], stmtSelectDayTmpls, *calendarEventID, i)
		if err != nil {
			log.Error("failed to get day tmpls by week day", "calendarEventID",
				*calendarEventID, "i", i, "error", err.Error())
			tx.Rollback()
			return
		}

		//get slots
		for j := range (*dayTmpls)[i] {
			err = ((*dayTmpls)[i])[j].Slots.Get(tx, ((*dayTmpls)[i])[j].ID, monday, i)
			if err != nil {
				return
			}
			if ((*dayTmpls)[i])[j].EndTime == "00:00" {
				((*dayTmpls)[i])[j].EndTime = "24:00"
			}
		}
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
