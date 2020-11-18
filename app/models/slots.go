package models

import (
	"fmt"
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
	Start time.Time `db:"start_time"`
	End   time.Time `db:"end_time"`
}

/*Insert a new slot. */
func (slot *Slot) Insert(v *revel.Validation, calendarEventID int) (data EMailData, err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	//check if all values are correct and the selected timespan is free
	if err = slot.validate(v, tx, calendarEventID); err != nil {
		return
	} else if v.HasErrors() {
		tx.Rollback()
		return
	}

	//insert Slot
	err = tx.Get(slot, stmtInsertSlot, slot.UserID, slot.DayTmplID,
		slot.Start, slot.End, time.Now())
	if err != nil {
		log.Error("failed to get Slot", "slot", *slot,
			"error", err.Error())
		tx.Rollback()
		return
	}

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
func (slot *Slot) validate(v *revel.Validation, tx *sqlx.Tx, calendarEventID int) (err error) {

	if !slot.Start.After(time.Now()) {
		v.ErrorKey("validation.calendar.event.slot.start.in.past")
		return
	}

	if !slot.Start.Before(slot.End) {
		v.ErrorKey("validation.calendar.event.slot.start.after.end")
		return
	}

	//ensure that the slot is in a valid day template of that calendar event
	dayTmpls := []DayTmpl{}
	weekday := int(slot.Start.Weekday()) - 1

	err = tx.Select(&dayTmpls, stmtGetDayTemplateFromWeekDay, weekday, calendarEventID)
	if err != nil {
		log.Error("failed to get day template", "calendarEventID", calendarEventID,
			"weekday", weekday, "error", err.Error())
		tx.Rollback()
		return
	}

	slotStartTime := CustomTime{}
	slotStartTime.Hour, slotStartTime.Min, _ = slot.Start.Clock()

	slotEndTime := CustomTime{}
	slotEndTime.Hour, slotEndTime.Min, _ = slot.End.Clock()

	tmplStartTime := CustomTime{}
	tmplEndTime := CustomTime{}

	isInTemplate := false
	indexDayTmpl := 0

	//find the day template in which the slot must occur
	for i := range dayTmpls {
		tmplStartTime.SetTime(dayTmpls[i].StartTime)
		tmplEndTime.SetTime(dayTmpls[i].EndTime)

		if tmplStartTime.Before(&slotStartTime) && (tmplEndTime.After(&slotEndTime) || tmplEndTime.Equals(&slotEndTime)) {
			isInTemplate = true
			indexDayTmpl = i
			slot.DayTmplID = dayTmpls[i].ID
			break
		}
	}

	if !isInTemplate {
		v.ErrorKey("validation.calendarEvent.noTemplateFitting")
		return
	}

	//slot starts at a valid interval section
	rem := slotStartTime.Sub(&tmplStartTime) % dayTmpls[indexDayTmpl].Interval
	if rem != 0 {
		v.ErrorKey("validation.calendarEvent.startTimeWrongStepDistance")
		return
	}

	//the slot length is a valid interval
	rem = slotEndTime.Sub(&tmplStartTime) % dayTmpls[indexDayTmpl].Interval
	if rem != 0 {
		v.ErrorKey("validation.calendarEvent.endTimeWrongStepDistance")
		return
	}

	//check if slot timespan is already occupied
	var slotUsed bool
	err = tx.Get(&slotUsed, stmtExistsOverlappingSlot, slot.Start,
		slot.End, slot.DayTmplID)
	if err != nil {
		log.Error("failed to get whether slot is already booked", "slotStartTimeValue",
			slotStartTime.Value, "slotEndTimeValue", slotEndTime.Value, "slotDayTmplID",
			slot.DayTmplID, "error", err.Error())
		tx.Rollback()
		return
	}

	fmt.Println(slotUsed)
	if slotUsed {
		v.ErrorKey("validation.calendarEvent.overlappingTimespanSlot")
		return
	}

	//check for exeption in that timespan
	//TODO: validate once we have exceptions
	/*
		slotUsed = false
		err = tx.Get(slotUsed, stmtExistsOverlappingExeption, slot.DayTmplID,
			slotStartTime.Value, slotEndTime.Value)
		if err != nil {
			log.Error("failed to get exist of Exeption",
				"error", err.Error())
		}

		if slotUsed {
			v.ErrorKey("validation.calendarEvent.overlappingTimespanExeption")
			return
		}
	*/

	return
}

/*Delete a Slot if it it is more than an hour away*/
func (slot *Slot) Delete(v *revel.Validation) (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	//get slot startTime by ID
	var startTime time.Time

	err = tx.Get(startTime, stmtGetSlotStartTime, slot.ID)
	if err != nil {
		log.Error("failed to get Slot", "slotID", slot.ID,
			"error", err.Error())
		tx.Rollback()
		return
	}

	//check if slot is more than an hour away
	var duration time.Duration = 1000000000 * 60 * 60

	if startTime.Sub(time.Now()) < duration {
		v.ErrorKey("validation.calendarEvent.deleteSlotToCloseToStart")
		tx.Commit()
		return
	}

	//delete day_tmpl
	if err = deleteByID("id", "slots", slot.ID, tx); err != nil {
		return
	}

	tx.Commit()
	return
}

const (
	stmtInsertSlot = `
		INSERT INTO slots (
				user_id, day_tmpl_id, start_time, end_time, created
			)
		VALUES (
				$1, $2, $3, $4, $5
		)
		RETURNING id
	`

	stmtSelectSlots = `
    SELECT id, user_id, day_tmpl_id, start_time, end_time, created
    FROM slots
    WHERE day_tmpl_id = $1
      AND start_time BETWEEN ($2) AND ($3)
		ORDER BY start_time ASC
  `

	stmtExistsOverlappingSlot = `
		SELECT EXISTS (
			SELECT true
			FROM slots
			WHERE day_tmpl_id = $3
				AND(
								($1 >= start_time AND $1 < end_time)
						OR 	($2 <= end_time AND $2 > start_time)
						OR 	(($1 <= start_time) AND ($2 >= end_time))
				)
		) AS slotUsed
	`

	stmtExistsOverlappingExeption = `
		SELECT EXISTS(
			SELECT true
			FROM day_tmpls d JOIN calendar_events e ON d.calendar_event_id = e.id
				JOIN calendar_exceptions ex ON e.id = ex.calendar_event_id
			WHERE d.id = $1
				AND (
								 ($2 BETWEEN (start_time) AND (end_time))
							OR ($3 BETWEEN (start_time) AND (end_time))
							OR (($2 <= start_time) AND ($3 >= end_time))
						)
		) AS slotUsed
	`

	stmtGetSlotStartTime = `
		SELECT start_time
		FROM slots
		WHERE id = $1
		AS startTime
	`
)
