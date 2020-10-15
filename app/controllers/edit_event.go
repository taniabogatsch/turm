package controllers

import (
	"database/sql"
	"errors"
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

	//TODO: do not allow the deletion of an event if users are enrolled in it
	//TODO: if users unsubscribed from this event, also remove them from the unsubscribed table

	event := models.Event{ID: ID}
	if err := event.Delete(); err != nil {
		return flashError(
			errDB, err, "/course/events?ID="+strconv.Itoa(courseID),
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

/*ChangeEnrollmentKey sets an enrollment key. */
func (c EditEvent) ChangeEnrollmentKey(ID int, key1, key2 string) revel.Result {

	c.Log.Debug("change enrollment key", "ID", ID, "key1", key1, "key2", key2)

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
		return flashError(
			errValidation, nil, "", c.Controller, "")
	}

	event := models.Event{ID: ID, EnrollmentKey: sql.NullString{Valid: true, String: key1}}
	if err := event.UpdateKey(); err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("event.key.change.success"))
	return c.Redirect(c.Session["currPath"])
}

/*DeleteEnrollmentKey of an event. */
func (c EditEvent) DeleteEnrollmentKey(ID int) revel.Result {

	c.Log.Debug("delete enrollment key", "ID", ID)

	//NOTE: the interceptor assures that the event ID is valid

	event := models.Event{ID: ID}
	err := event.Update("enrollment_key", sql.NullString{"", false})

	if err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("event.key.delete.success"))
	return c.Redirect(c.Session["currPath"])
}

/*CreateDayTemplate is used for crating a repeatable blueprint if a day*/
func (c EditEvent) CreateDayTemplate(ID int, startTime, endTime string, intervall int, dayOfWeek int) revel.Result {
	c.Log.Debug("create a day template on course", "CourseID", ID,
		"startTime", startTime, "endTime", endTime, "dayOfWeek", dayOfWeek)

	time := models.CustomTime{}
	isValidTime1 := time.SetTime(startTime)
	isValidTime2 := time.SetTime(endTime)

	if isValidTime1 == false || isValidTime2 == false {
		err := errors.New("Inserted value is not a valid Time ")
		return flashError(
			errAuth, err, "", c.Controller, "")
	}

	dayT := models.DayTmpl{CalendarEventID: ID, StartTime: startTime,
		EndTime: endTime, Intervall: intervall, DayOfWeek: dayOfWeek}

	err := dayT.Insert()
	if err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("DayTemplate.new.success",
		dayT.StartTime,
		dayT.EndTime,
		dayT.ID,
	))
	return c.Redirect(c.Session["currPath"])
}
