package controllers

import (
	"database/sql"
	"errors"
	"strings"
	"turm/app/models"

	"github.com/revel/revel"
)

/*Delete event data.
- Roles: creator and editors of the course of the event */
func (c EditEvent) Delete(ID int) revel.Result {

	c.Log.Debug("delete event", "ID", ID)

	//NOTE: the interceptor assures that the event ID is valid

	//TODO: do not allow the deletion of an event if users are enrolled in it
	//TODO: if users unsubscribed from this event, also remove them from the unsubscribed table

	event := models.Event{ID: ID}
	err := event.Delete()
	if err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("event.delete.success",
		ID,
	))
	return c.Redirect(c.Session["currPath"])
}

/*NewMeeting creates a new blank meeting in an event.
- Roles: creator and editors of the course of the event */
func (c EditEvent) NewMeeting(ID int, option models.MeetingInterval) revel.Result {

	c.Log.Debug("create a new meeting", "ID", ID, "option", option)

	//NOTE: the interceptor assures that the event ID is valid

	if option > models.ODD || option < models.SINGLE {
		c.Validation.ErrorKey("validation.invalid.option")
	}

	if c.Validation.HasErrors() {
		return flashError(
			errValidation, nil, "", c.Controller, "")
	}

	meeting := models.Meeting{EventID: ID, MeetingInterval: option}
	err := meeting.NewBlank()
	if err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("meeting.new.success",
		meeting.ID,
	))
	return c.Redirect(c.Session["currPath"])
}

/*ChangeCapacity changes the capacity of an event.
- Roles: creator and editors of the course of the event */
func (c EditEvent) ChangeCapacity(ID int, fieldID string, value int) revel.Result {

	c.Log.Debug("change capacity", "ID", ID, "fieldID", fieldID, "value", value)

	//NOTE: the interceptor assures that the event ID is valid

	c.Validation.Check(value,
		revel.Min{1},
		revel.Max{1000000},
	).MessageKey("validation.invalid.int")

	if c.Validation.HasErrors() {
		return flashError(
			errValidation, nil, "", c.Controller, "")
	}

	if fieldID != "capacity" {
		return flashError(
			errContent,
			errors.New("invalid column value"),
			"", c.Controller, "")
	}

	event := models.Event{ID: ID}
	err := event.Update(fieldID, value)
	if err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("event.capacity.change.success",
		event.Capacity,
	))
	return c.Redirect(c.Session["currPath"])
}

/*ChangeText changes the text of the provided column.
- Roles: creator and editors of the course of the event */
func (c EditEvent) ChangeText(ID int, fieldID, value string) revel.Result {

	c.Log.Debug("change text value", "ID", ID, "fieldID", fieldID, "value", value)

	value = strings.TrimSpace(value)
	valid := (value != "")

	//NOTE: the interceptor assures that the event ID is valid
	if value != "" || fieldID == "title" {

		c.Validation.Check(value,
			revel.MinSize{3},
			revel.MaxSize{255},
		).MessageKey("validation.invalid.text.short")

		if c.Validation.HasErrors() {
			return flashError(
				errValidation, nil, "", c.Controller, "")
		}
	}

	if fieldID != "title" && fieldID != "annotation" {
		return flashError(
			errContent,
			errors.New("invalid column value"),
			"", c.Controller, "")
	}

	event := models.Event{ID: ID}
	err := event.Update(fieldID, sql.NullString{value, valid})

	if err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
	}

	if valid {
		c.Flash.Success(c.Message("event."+fieldID+".change.success",
			value,
		))
	} else {
		c.Flash.Success(c.Message("event." + fieldID + ".delete.success"))
	}
	return c.Redirect(c.Session["currPath"])
}

/*ChangeWaitlist toggles the waitlist setting of an event.
- Roles: creator and editors of the course of the event */
func (c EditEvent) ChangeWaitlist(ID int, option bool) revel.Result {

	c.Log.Debug("update waitlist setting", "ID", ID, "option", option)

	//NOTE: the interceptor assures that the event ID is valid

	//TODO: only allow to toggle the waitlist setting if it is not true
	//and there are users enrolled in the event

	event := models.Event{ID: ID, HasWaitlist: option}
	if err := event.Update("has_waitlist", event.HasWaitlist); err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("event.waitlist.change.success"))
	return c.Redirect(c.Session["currPath"])
}
