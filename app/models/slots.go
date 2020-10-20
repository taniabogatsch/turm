package models

import (
	"time"
	"turm/app"

	"github.com/jmoiron/sqlx"
	"github.com/revel/revel"
)

/*Slots booked at a specific day within StartTime and EndTime of a day template. */
type Slots []Slot

/*Slot is a booked timespan on an specific date. */
type Slot struct {
	ID        int    `db:"id"`
	DayTmplID int    `db:"day_tmpl_id"`
	UserID    int    `db:"user_id"`
	Created   string `db:"created"`

	//date + time
	StartTimestamp time.Time `db:"start_time"`
	EndTimestamp   time.Time `db:"end_time"`
}

/*Insert a new slot. */
func (slot *Slot) Insert(v *revel.Validation) (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	if slot.validate(v, tx); v.HasErrors() {
		tx.Rollback()
		return
	}

	//TODO @Marco

	tx.Commit()
	return
}

/*Get all slots of a day template. Monday specifies the week for which all slots
must be loaded and weekday specifies the day. */
func (slots *Slots) Get(tx *sqlx.Tx, dayTmplID int, monday time.Time, weekday int) (err error) {

	//[startTime, endTime] is [day 00:00 - day 24:00]
	startTime := monday.AddDate(0, 0, weekday-1)
	endTime := startTime.Add(1000000000 * 60 * 60 * 24)

	err = tx.Select(slots, stmtSelectSlots, dayTmplID, startTime, endTime)
	if err != nil {
		log.Error("failed to get slots of day template", "DayTmplID", dayTmplID,
			"monday", monday, "weekday", weekday, "error", err.Error())
		tx.Rollback()
	}
	return
}

//validate the slot struct
func (slot *Slot) validate(v *revel.Validation, tx *sqlx.Tx) {

	startInPast := slot.StartTimestamp.After(time.Now())
	v.Check(startInPast).
		MessageKey("validation.calendarEvent.startInPast")

	startAfterEndTime := slot.StartTimestamp.Before(slot.EndTimestamp)
	v.Check(startAfterEndTime).
		MessageKey("validation.calendarEvent.startAfterEndTime")

	//chek if startTime and endTime is on same date
	y1, m1, d1 := slot.StartTimestamp.Date()
	y2, m2, d2 := slot.EndTimestamp.Date()
	v.Check(y1 == y2 && m1 == m2 && d1 == d2).
		MessageKey("validation.calendarEvent.startOtherDayThanEnd")

	dayTmpls := []DayTmpl{}

	weekday := int(slot.StartTimestamp.Weekday())

	err := tx.Select(dayTmpls, stmtGetDayTemplateFromWeekDay, weekday)
	if err != nil {
		log.Error("failed to get DayTemolate", "weekday", weekday,
			"error", err.Error())
	}

	slotStartTime := Custom_time{}
	slotStartTime.Hour, slotStartTime.Min, _ = slot.StartTimestamp.Clock()

	slotEndTime := Custom_time{}
	slotEndTime.Hour, slotEndTime.Min, _ = slot.EndTimestamp.Clock()

	tmplStartTime := Custom_time{}
	tmplEndTime := Custom_time{}

	isInTemplate := false
	indexDayTmpl := 0

	//look for a DayTemplate where the slot is within its time intervall
	for i := range dayTmpls {
		tmplStartTime.SetTime(dayTmpls[i].StartTime)
		tmplEndTime.SetTime(dayTmpls[i].EndTime)

		if tmplStartTime.Before(slotStartTime) && tmplEndTime.After(slotEndTime) {
			isInTemplate = true
			indexDayTmpl = i
			break
		}
	}

	v.Check(isInTemplate).MessageKey("validation.calendarEvent.noTemplateFitting")

	//schrittweite prüfen
	//types ok?
	intervalSteps := float64(slotStartTime.Sub(tmplStartTime) / dayTmpls[indexDayTmpl].Interval)
	startWrongStepDistance := (intervalSteps - float64(int(intervalSteps))) == 0
	v.Check(startWrongStepDistance).MessageKey("validation.calendarEvent.startTimeWrongStepDistance")

	intervalSteps = float64(slotEndTime.Sub(tmplStartTime) / dayTmpls[indexDayTmpl].Interval)
	endWrongStepDistance := (intervalSteps - float64(int(intervalSteps))) == 0
	v.Check(endWrongStepDistance).MessageKey("validation.calendarEvent.endTimeWrongStepDistance")

	//TODO: schon besetzt?
}

//TODO: delete (unsubscribe)

const (
	stmtSelectSlots = `
    SELECT id, user_id, day_tmpl_id, start_time, end_time, created
    FROM slots
    WHERE day_tmpl_id = $1
      AND start_time BETWEEN ($2) AND ($3);
  `
)
