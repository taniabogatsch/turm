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
	ID        int `db:"id"`
	DayTmplID int `db:"day_tmpl_id"`
	UserID    int `db:"user_id"`

	//date + time
	Start time.Time `db:"start_time"`
	End   time.Time `db:"end_time"`

	//used for participants management
	User     User
	StartStr string `db:"start_str"`
	EndStr   string `db:"end_str"`
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

	//insert slot
	err = tx.Get(slot, stmtInsertSlot, slot.UserID, slot.DayTmplID,
		slot.Start, slot.End)
	if err != nil {
		log.Error("failed to insert slot", "slot", *slot, "error", err.Error())
		tx.Rollback()
		return
	}

	//get e-mail data
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

	tx.Commit()
	return
}

/*Get all slots of a day template. Monday specifies the week for which all slots
must be loaded and weekday specifies the day. */
func (slots *Slots) Get(tx *sqlx.Tx, dayTmplID int, monday time.Time, weekday int) (err error) {

	//set time of monday to 00:00
	monday = time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, monday.Location())

	//[startTime, endTime] is [day 00:00 - day 24:00]
	startTime := monday.AddDate(0, 0, weekday)
	endTime := startTime.Add(1000000000 * 60 * 60 * 24)

	err = tx.Select(slots, stmtSelectSlots, dayTmplID, startTime, endTime, app.TimeZone)
	if err != nil {
		log.Error("failed to get slots of day template on specific day", "dayTmplID", dayTmplID,
			"startTime", startTime, "endTime", endTime, "weekday", weekday, "error", err.Error())
		tx.Rollback()
	}
	return
}

/*GetAll slots of a day template. */
func (slots *Slots) GetAll(tx *sqlx.Tx, dayTmplID int) (err error) {

	//get slot data for validation
	err = tx.Select(slots, stmtSelectAllSlotsOfDayTemplate, dayTmplID)
	if err != nil {
		log.Error("failed to get all slots of a day template", "dayTmplID", dayTmplID,
			"error", err.Error())
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

	//the weekday time.Weekday function is 1 greater than our weekDay
	//to compare we have to subtract one (so sundays 0 has to become 7)
	if weekday == -1 {
		weekday = 6
	}

	err = tx.Select(&dayTmpls, stmtSelectDayTemplatesFromWeekDay, weekday, calendarEventID)
	if err != nil {
		log.Error("failed to get day template from week day", "calendarEventID",
			calendarEventID, "weekday", weekday, "error", err.Error())
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

		if tmplStartTime.Before(&slotStartTime) && (tmplEndTime.After(&slotEndTime) ||
			tmplEndTime.Equals(&slotEndTime)) {
			isInTemplate = true
			indexDayTmpl = i
			slot.DayTmplID = dayTmpls[i].ID
			break
		}
	}

	if !isInTemplate {
		v.ErrorKey("validation.calendar.event.no.fitting.tmpl")
		return
	}

	//slot starts at a valid interval section
	rem := slotStartTime.Sub(&tmplStartTime) % dayTmpls[indexDayTmpl].Interval
	if rem != 0 {
		v.ErrorKey("validation.calendar.event.start.wrong.step.distance")
		return
	}

	//the slot length is a valid interval
	rem = slotEndTime.Sub(&tmplStartTime) % dayTmpls[indexDayTmpl].Interval
	if rem != 0 {
		v.ErrorKey("validation.calendar.event.start.wrong.step.distance")
		return
	}

	//check if slot timespan is already occupied
	var slotUsed bool
	err = tx.Get(&slotUsed, stmtExistsOverlappingSlot, slot.Start,
		slot.End, slot.DayTmplID)
	if err != nil {
		log.Error("failed to get whether slot is already booked", "slotStart",
			slot.Start, "slotEnd", slot.End, "slotDayTmplID", slot.DayTmplID,
			"error", err.Error())
		tx.Rollback()
		return
	}

	if slotUsed {
		v.ErrorKey("validation.calendar.event.slots.overlap")
		return
	}

	//check for an exception in that timespan
	slotInException := false
	err = tx.Get(&slotInException, stmtExistsOverlappingExeption, calendarEventID,
		slot.Start, slot.End)
	if err != nil {
		log.Error("failed to get if the slot overlaps an exception", "calendarEventID",
			calendarEventID, "slotStart", slot.Start, "slotEnd", slot.End,
			"error", err.Error())
		tx.Rollback()
		return
	}

	if slotInException {
		v.ErrorKey("validation.calendar.event.slot.overlaps.exception")
		return
	}

	return
}

/*Delete a slot if it it is more than an hour away. */
func (slot *Slot) Delete(v *revel.Validation) (data EMailData, err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	//get slot startTime by ID
	var startTime time.Time
	err = tx.Get(&startTime, stmtGetSlotStartTime, slot.ID)
	if err != nil {
		log.Error("failed to get slot start time", "slotID", slot.ID,
			"error", err.Error())
		tx.Rollback()
		return
	}

	//check if slot is more than an hour away
	var duration time.Duration = 1000000000 * 60 * 60
	if startTime.Sub(time.Now()) < duration {
		v.ErrorKey("validation.calendar.event.slot.unsubscribe.end")
		tx.Rollback()
		return
	}

	//get e-mail data
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

	//delete slot
	_, err = tx.Exec(stmtDeleteSlot, slot.ID, slot.UserID)
	if err != nil {
		log.Error("failed to delete the slot", "slot", *slot,
			"error", err.Error())
		tx.Rollback()
		return
	}

	tx.Commit()
	return
}

/*DeleteManual manually deletes a slot. */
func (slot *Slot) DeleteManual() (data EMailData, err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	err = tx.Get(&slot.UserID, stmtGetUserID, slot.ID)
	if err != nil {
		log.Error("failed to get slot data for e-mail", "slotID", slot.ID,
			"error", err.Error())
		tx.Rollback()
		return
	}

	//get e-mail data
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

	//delete slot
	_, err = tx.Exec(stmtDeleteSlotManually, slot.ID)
	if err != nil {
		log.Error("failed to delete the slot", "slot", *slot,
			"error", err.Error())
		tx.Rollback()
		return
	}

	tx.Commit()
	return
}

/*BelongsToEvent checks if a slot belongs to an event*/
func (slot *Slot) BelongsToEvent(eventID int) (belongs bool, err error) {

	err = app.Db.Get(&belongs, stmtSlotBelongsToEvent, slot.ID, eventID)
	if err != nil {
		log.Error("failed to get if the slot belongs to the specified event", "slot", *slot,
			"event ID", eventID, "error", err.Error())
	}
	return
}

const (
	stmtInsertSlot = `
		INSERT INTO slots (
			user_id, day_tmpl_id, start_time, end_time)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	stmtSelectSlots = `
    SELECT id, user_id, day_tmpl_id, start_time, end_time,
		 TO_CHAR (start_time AT TIME ZONE $4, 'YYYY-MM-DD HH24:MI') AS start_str,
		 TO_CHAR (end_time AT TIME ZONE $4, 'YYYY-MM-DD HH24:MI') AS end_str
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
				AND (
					($1 >= start_time AND $1 < end_time)
					OR 	($2 <= end_time AND $2 > start_time)
					OR 	(($1 <= start_time) AND ($2 >= end_time))
				)
		) AS slot_used
	`

	stmtExistsOverlappingExeption = `
		SELECT EXISTS(
			SELECT true
			FROM calendar_exceptions
			WHERE calendar_event_id = $1
				AND (
					($2 >= exception_start AND $2 < exception_end)
					OR ($3 <= exception_end 	AND $3 > exception_start)
					OR (($2 <= exception_start) AND ($3 >= exception_end))
				)
		) AS slot_in_exception
	`

	stmtGetSlotStartTime = `
		SELECT start_time
		FROM slots
		WHERE id = $1
	`

	stmtSelectAllSlotsOfDayTemplate = `
		SELECT id, user_id, day_tmpl_id, start_time, end_time,
			start_time AS start_str, end_time AS end_str
		FROM slots
		WHERE day_tmpl_id = $1
		ORDER BY start_time ASC
	`

	stmtGetSlotEMailData = `
		SELECT c.title AS course_title, e.title AS event_title, c.id AS course_id,
			TO_CHAR (s.start_time AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') AS start,
			TO_CHAR (s.end_time AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') AS end
		FROM slots s JOIN day_templates d ON s.day_tmpl_id = d.id
			JOIN calendar_events e ON e.id = d.calendar_event_id
			JOIN courses c ON c.id = e.course_id
		WHERE s.id = $1
	`

	stmtDeleteSlot = `
		DELETE FROM slots
		WHERE id = $1
			AND user_id = $2
	`

	stmtDeleteSlotManually = `
		DELETE FROM slots
		WHERE id = $1
	`

	stmtSlotBelongsToEvent = `
		SELECT EXISTS (
			SELECT true
			FROM slots s JOIN day_templates t ON s.day_tmpl_id = t.id
			WHERE t.calendar_event_id = $2
				AND s.id = $1
		) AS belongs
	`

	stmtGetUserID = `
	SELECT user_id
	FROM slots
	WHERE id = $1
	`
)
