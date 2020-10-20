package models

import (
	"database/sql"
	"time"
	"turm/app"

	"github.com/jmoiron/sqlx"
	"github.com/revel/revel"
)

/*ScheduleEntryType is a type for encoding different schedule entries. */
type ScheduleEntryType int

const (
	//EMPTY is for no entry
	EMPTY ScheduleEntryType = iota
	//SLOT is for slots
	SLOT
	//EXCEPTION is for exceptions
	EXCEPTION
)

func (s ScheduleEntryType) String() string {
	return [...]string{"empty", "slot", "exception"}[s]
}

/*DayTmpls of a week for each day. */
type DayTmpls []TmplsOfDay

/*TmplsOfDay contains all day templates of a specific week day. */
type TmplsOfDay []DayTmpl

/*DayTmpl is a section of a week day (Monday - Sunday). */
type DayTmpl struct {
	ID               int            `db:"id"`
	CalendarEventID  int            `db:"calendar_event_id"`
	StartTime        string         `db:"start_time"`
	EndTime          string         `db:"end_time"`
	Interval         int            `db:"interval"`
	DayOfWeek        int            `db:"day_of_week"` //must be an integer between [0, 6]
	Active           bool           `db:"active"`
	DeactiavtionDate sql.NullString `db:"deactivation_date"`

	Slots      Slots
	Exceptions Exceptions

	Schedule Schedule
}

/*Schedule is a helper struct to display a day template at the front end. */
type Schedule []ScheduleEntry

/*ScheduleEntry containing all information to print a section of a day template. */
type ScheduleEntry struct {
	StartTime string
	EndTime   string //should be the same as the subsequent start time
	Type      ScheduleEntryType
}

/*Insert a DayTemplate. */
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
	}

	tx.Commit()
	return
}

//validate a day template.
func (tmpl *DayTmpl) validate(v *revel.Validation, tx *sqlx.Tx) {

	time := CustomTime{}
	isValidTime1 := time.SetTime(tmpl.StartTime)
	isValidTime2 := time.SetTime(tmpl.EndTime)

	if isValidTime1 == false || isValidTime2 == false {
		v.ErrorKey("validation.invalid.timestamp")
	}

	//TODO!
}

/*Update a day tmpl. */
func (tmpl *DayTmpl) Update(v *revel.Validation) (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	if tmpl.validate(v, tx); v.HasErrors() {
		tx.Rollback()
		return
	}

	//TODO: update

	tx.Commit()
	return
}

/*Get all day templates of a CalendarEvent. */
func (dayTmpls *DayTmpls) Get(tx *sqlx.Tx, calendarEventID *int, monday time.Time) (err error) {

	//init a slice for each week day
	*dayTmpls = append(*dayTmpls, TmplsOfDay{}, TmplsOfDay{}, TmplsOfDay{},
		TmplsOfDay{}, TmplsOfDay{}, TmplsOfDay{}, TmplsOfDay{})

	//iterate week days and get day templates of each day
	for i := 0; i < 7; i++ {
		err = tx.Select(&(*dayTmpls)[i], stmtSelectDayTmpls, *calendarEventID, i)
		if err != nil {
			log.Error("failed to get day tmpls by week day", "calendarEventID",
				*calendarEventID, "i", i, "error", err.Error())
			tx.Rollback()
			return
		}

		for j := range (*dayTmpls)[i] {
			//get slots
			err = ((*dayTmpls)[i])[j].Slots.Get(tx, ((*dayTmpls)[i])[j].ID, monday, i)
			if err != nil {
				return
			}
			//get exceptions
			err = ((*dayTmpls)[i])[j].Exceptions.Get(tx, monday, i)
			if err != nil {
				return
			}
		}
	}

	//TODO: prepare for front end

	return
}

const (
	stmtInsertDayTemplate = `
    INSERT INTO day_templates (
        calendar_event_id, start_time, end_time, intervall, day_of_week
      )
    VALUES (
        $1, $2, $3, $4, $5
    )
    RETURNING id
  `

	stmtGetDayTemplateFromWeekDay = `
    SELECT id, start_time, end_time, intevall
    FROM day_templates
    WHERE day_of_week = $1
      AND active = true
  `

	stmtSelectDayTmpls = `
    SELECT id, calendar_event_id, start_time, end_time, interval,
      day_of_week, active, deactivation_date
    FROM day_templates
    WHERE calendar_event_id = $1
      AND active = true
      AND day_of_week = $2
    ORDER BY start_time ASC
  `
)
