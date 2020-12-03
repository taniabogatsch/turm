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
	Type      ScheduleEntryType
}

/*Insert a day template. */
func (tmpl *DayTmpl) Insert(v *revel.Validation) (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
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

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	//validate new DayTemplate
	tmpl.validate(v, tx)

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

		//check if slot is not within the new time
		if !((tmplStartTime.Before(&slotStartTime) && (tmplEndTime.After(&slotEndTime) ||
			tmplEndTime.Equals(&slotEndTime))) || int(slot.Start.Weekday()) != tmpl.DayOfWeek) {

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

			//create user for email if slot isnt in past
			if slot.End.Before(time.Now()) {

				user := User{ID: slot.UserID}
				if err = user.Get(tx); err != nil {
					return
				}
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

	//TODO: remove all slots that are no longer in day template time,
	//or that no longer match the interval
	//- if slot in past: no e-mail to users
	//- else send e-mail

	//TODO: update times, interval, day

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

	if !start.Before(&end) {
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
				AND id != $4
				AND (
					($1 >= start_time AND $1 < end_time)
					OR 	($2 <= end_time AND $2 > start_time)
					OR 	(($1 <= start_time) AND ($2 >= end_time))
				)
		) AS overlaps
	`
)
