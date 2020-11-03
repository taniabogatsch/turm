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
	//BLOCKED is for Timeslots between
	BLOCKED
)

func (s ScheduleEntryType) String() string {
	return [...]string{"empty", "slot", "exception", "blocked"}[s]
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

	Slots Slots
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

	//check for valid times
	startTime := Custom_time{}
	isValidTime1 := startTime.SetTime(tmpl.StartTime)

	endTime := Custom_time{}
	isValidTime2 := endTime.SetTime(tmpl.EndTime)

	v.Check(isValidTime1 == false || isValidTime2 == false).
		MessageKey("validation.invalid.timestamp")

	//check startTime before endTime
	v.Check(startTime.Before(endTime)).
		MessageKey("validation.calendarEvent.startInPast")

	//check step distance
	intervalSteps := float64(startTime.Sub(endTime) / tmpl.Interval)
	startWrongStepDistance := (intervalSteps - float64(int(intervalSteps))) == 0
	v.Check(startWrongStepDistance).MessageKey("validation.calendarEvent.endTimeWrongStepDistance")

	//check if template collides with other template on that day
	var timeOverlapTmpl bool

	err := tx.Get(timeOverlapTmpl, stmtGetExistsOverlappingDayTmpl, startTime.Value, endTime.Value, tmpl.CalendarEventID)
	if err != nil {
		log.Error("failed to get DayTemolates",
			"error", err.Error())
	}

	v.Check(!timeOverlapTmpl).
		MessageKey("validation.calendarEvent.overlappingTimespanDayTmpl")
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

	//validate checks for time -> can collide with itselfe when changing time
	//what happens to slots when changing Timespan of template
	//TODO: update

	tx.Commit()
	return
}

/*Delete a day Template if it has no slots*/
func (tmpl *DayTmpl) Delete(v *revel.Validation) (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	//check if DayTmpl has no slots
	var notEmpty bool
	err = tx.Get(notEmpty, stmtExistSlots, tmpl.ID)
	if err != nil {
		log.Error("failed to get DayTemolate", "templateID", tmpl.ID,
			"error", err.Error())
		tx.Rollback()
		return
	}

	if notEmpty {
		v.ErrorKey("validation.calendarEvent.deleteDayTemplateNotEmpty")
		tx.Commit()
		return
	}

	//delete day_tmpl
	if err = deleteByID("id", "calendar_events", tmpl.ID, tx); err != nil {
		return
	}

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

			//TODO: fill scedule for that day tmpl

		}
	}

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
		ORDER BY start_time ASC
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

	stmtGetExistsOverlappingDayTmpl = `
		SELECT EXISTS(
			SELECT true
			FROM day_templates
			WHERE $3 = calendar_event_id
				AND active = true
				AND (
									($1 BETWEEN (start_time) AND (end_time))
							OR 	($2 BETWEEN (start_time) AND (end_time))
							OR  (($1 <= start_time) AND ($2 >= end_time))
						)
		) AS timeOverlapTmpl
	`

	stmtExistSlots = `
		SELECT EXISTS (
			SELECT true
			FROM slots
			WHERE day_tmpl_id = $1
		) AS notEmpty
	`
)
