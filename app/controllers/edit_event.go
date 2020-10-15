package controllers

import (
	"database/sql"
	"strconv"
	"strings"
	"turm/app/models"

	"github.com/revel/revel"
)

/*Delete event data.
- Roles: creator and editors of the course of the event */
func (c EditEvent) Delete(ID, courseID int) revel.Result {

	c.Log.Debug("delete event", "ID", ID)

	//NOTE: the interceptor assures that the event ID is valid

	event := models.Event{ID: ID}
	if err := event.Delete(c.Validation); err != nil {
		return flashError(
			errDB, err, "/course/events?ID="+strconv.Itoa(courseID),
			c.Controller, "")
	} else if c.Validation.HasErrors() {
		return flashError(
			errValidation, nil, "/course/events?ID="+strconv.Itoa(courseID),
			c.Controller, "")
	}

	c.Flash.Success(c.Message("event.delete.success", ID))
	return c.Redirect(Course.Events, courseID)
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
			errValidation, nil, "/course/meetings?ID="+strconv.Itoa(ID),
			c.Controller, "")
	}

	meeting := models.Meeting{EventID: ID, MeetingInterval: option}
	if err := meeting.NewBlank(); err != nil {
		return flashError(
			errDB, err, "/course/meetings?ID="+strconv.Itoa(ID),
			c.Controller, "")
	}

	c.Flash.Success(c.Message("meeting.new.success", meeting.ID))
	return c.Redirect(Course.Meetings, ID)
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
		return c.RenderJSON(
			response{Status: INVALID, Msg: getErrorString(c.Validation.Errors)})
	}

	if fieldID != "capacity" {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message("error.undefined")})
	}

	event := models.Event{ID: ID}
	err := event.Update(fieldID, value)
	if err != nil {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message(errDB.String())})
	}

	msg := c.Message("event.capacity.change.success", event.Capacity)
	return c.RenderJSON(
		response{Status: SUCCESS, Msg: msg, FieldID: fieldID, Value: strconv.Itoa(value), ID: ID})
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
			return c.RenderJSON(
				response{Status: INVALID, Msg: getErrorString(c.Validation.Errors)})
		}
	}

	if fieldID != "title" && fieldID != "annotation" {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message("error.undefined")})
	}

	event := models.Event{ID: ID}
	err := event.Update(fieldID, sql.NullString{value, valid})
	if err != nil {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message(errDB.String())})
	}

	msg := c.Message("event." + fieldID + ".delete.success")
	if valid {
		msg = c.Message("event."+fieldID+".change.success", value)
	}

	return c.RenderJSON(
		response{Status: SUCCESS, Msg: msg, FieldID: fieldID, Value: value, ID: ID})
}

/*ChangeBool toggles the provided boolean value of an event.
- Roles: creator and editors of the course of the event */
func (c EditEvent) ChangeBool(ID int, listType string, option bool) revel.Result {

	c.Log.Debug("update bool", "ID", ID, "listType", listType, "option", option)

	//NOTE: the interceptor assures that the event ID is valid

	//TODO: only allow to toggle waitlist to false if there are no users at the wait list

	if listType != "has_waitlist" {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message("error.undefined")})
	}

	event := models.Event{ID: ID}
	if err := event.Update(listType, option); err != nil {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message(errDB.String())})
	}

	msg := c.Message("event.waitlist.change.success")
	return c.RenderJSON(
		response{Status: SUCCESS, Msg: msg, FieldID: listType, Valid: option, ID: ID})
}

/*ChangeEnrollmentKey sets an enrollment key.
- Roles: creator and editors of the course of the event */
func (c EditEvent) ChangeEnrollmentKey(ID int, key1, key2, fieldID string) revel.Result {

	c.Log.Debug("change enrollment key", "ID", ID, "key1", key1, "key2", key2, "fieldID", fieldID)

	//NOTE: the interceptor assures that the event ID is valid

	key1 = strings.TrimSpace(key1)
	key2 = strings.TrimSpace(key2)
	if key1 == key2 {
		c.Validation.Check(key1,
			revel.MinSize{3},
			revel.MaxSize{511}).
			MessageKey("validation.invalid.keys")
	} else {
		c.Validation.ErrorKey("validation.invalid.keys")
	}

	if c.Validation.HasErrors() {
		return c.RenderJSON(
			response{Status: INVALID, Msg: getErrorString(c.Validation.Errors)})
	}

	event := models.Event{ID: ID, EnrollmentKey: sql.NullString{Valid: true, String: key1}}
	if err := event.UpdateKey(); err != nil {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message(errDB.String())})
	}

	msg := c.Message("event.key.change.success")
	return c.RenderJSON(
		response{Status: SUCCESS, Msg: msg, FieldID: fieldID, Value: "key", ID: ID})
}

/*DeleteEnrollmentKey of an event.
- Roles: creator and editors of the course of the event */
func (c EditEvent) DeleteEnrollmentKey(ID int) revel.Result {

	c.Log.Debug("delete enrollment key", "ID", ID)

	//NOTE: the interceptor assures that the event ID is valid

	event := models.Event{ID: ID}
	err := event.Update("enrollment_key", sql.NullString{"", false})

	if err != nil {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message(errDB.String())})
	}

	msg := c.Message("event.key.delete.success")
	return c.RenderJSON(
		response{Status: SUCCESS, Msg: msg, FieldID: "enrollment_key", ID: ID})
}
