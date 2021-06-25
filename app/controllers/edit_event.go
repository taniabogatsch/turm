package controllers

import (
	"database/sql"
	"strconv"
	"strings"
	"turm/app/models"

	"github.com/revel/revel"
)

/*Delete event.
- Roles: creator and editors of the course of the event */
func (c EditEvent) Delete(ID, courseID int) revel.Result {

	c.Log.Debug("delete event", "ID", ID, "courseID", courseID)
	c.Session["lastURL"] = c.Request.URL.String()

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

/*Duplicate event.
- Roles: creator and editors of the course of the event */
func (c EditEvent) Duplicate(ID, courseID int) revel.Result {

	c.Log.Debug("duplicate event", "ID", ID, "courseID", courseID)
	c.Session["lastURL"] = c.Request.URL.String()

	//NOTE: the interceptor assures that the event ID is valid

	event := models.Event{ID: ID, CourseID: courseID}
	if err := event.Duplicate(nil); err != nil {
		return flashError(
			errDB, err, "/course/events?ID="+strconv.Itoa(courseID),
			c.Controller, "")
	}

	c.Flash.Success(c.Message("event.duplicate.success", ID))
	return c.Redirect(Course.Events, courseID)
}

/*NewMeeting creates a new blank meeting that is part of an event.
- Roles: creator and editors of the course of the event */
func (c EditEvent) NewMeeting(ID int, option models.MeetingInterval,
	conf models.EditEMailConfig) revel.Result {

	c.Log.Debug("create a new meeting", "ID", ID, "option", option,
		"conf", conf)
	c.Session["lastURL"] = c.Request.URL.String()

	//NOTE: the interceptor assures that the event ID is valid

	if option > models.ODD || option < models.SINGLE {
		c.Validation.ErrorKey("validation.invalid.option")
	}

	if c.Validation.HasErrors() {
		return flashError(
			errValidation, nil, "/course/meetings?ID="+strconv.Itoa(ID),
			c.Controller, "")
	}

	conf.ID = ID
	conf.IsEvent = true

	meeting := models.Meeting{EventID: ID, MeetingInterval: option}
	if err := meeting.NewBlank(&conf); err != nil {
		return flashError(
			errDB, err, "/course/meetings?ID="+strconv.Itoa(ID),
			c.Controller, "")
	}

	//if the course is active, send notification e-mail
	conf.Field = "email.edit.new.meeting"
	if err := sendEMailsEdit(c.Controller, &conf); err != nil {
		return flashError(errEMail, err, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("meeting.new.success", meeting.ID))
	return c.Redirect(Course.Meetings, ID)
}

/*ChangeCapacity changes the capacity of an event.
- Roles: creator and editors of the course of the event */
func (c EditEvent) ChangeCapacity(ID int, fieldID string, value int) revel.Result {

	c.Log.Debug("change capacity", "ID", ID, "fieldID", fieldID, "value", value)
	c.Session["lastURL"] = c.Request.URL.String()

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
	users, err := event.Update(nil, fieldID, value, nil)
	if err != nil {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message(errDB.String())})
	}

	//auto enroll users from wait list if the capacity is changed
	for _, user := range users {

		err = sendEMail(c.Controller, &user,
			"email.subject.from.wait.list",
			"fromWaitlist")

		if err != nil {
			return flashError(errEMail, err, "", c.Controller, user.User.EMail)
		}
	}

	msg := c.Message("event.capacity.change.success", event.Capacity)
	return c.RenderJSON(
		response{Status: SUCCESS, Msg: msg, FieldID: fieldID,
			Value: strconv.Itoa(value), ID: ID, Fullness: strconv.Itoa(event.Fullness)})
}

/*ChangeText changes the text of the provided column.
- Roles: creator and editors of the course of the event */
func (c EditEvent) ChangeText(ID int, fieldID, value string,
	conf models.EditEMailConfig) revel.Result {

	c.Log.Debug("change text value", "ID", ID, "fieldID", fieldID, "value", value,
		"conf", conf)
	c.Session["lastURL"] = c.Request.URL.String()

	value = strings.TrimSpace(value)
	valid := (value != "")

	//NOTE: the interceptor assures that the event ID is valid
	if value != "" || fieldID == colTitle {

		models.ValidateLength(&value, "validation.invalid.text.short",
			3, 255, c.Validation)

		if c.Validation.HasErrors() {
			return c.RenderJSON(
				response{Status: INVALID, Msg: getErrorString(c.Validation.Errors)})
		}
	}

	if fieldID != colTitle && fieldID != colAnnotation {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message("error.undefined")})
	}

	conf.ID = ID
	conf.IsEvent = true

	event := models.Event{ID: ID}
	_, err := event.Update(nil, fieldID, sql.NullString{
		String: value,
		Valid:  valid,
	}, &conf)
	if err != nil {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message(errDB.String())})
	}

	//if the course is active, send notification e-mail
	conf.Field = "email.edit." + fieldID
	if err = sendEMailsEdit(c.Controller, &conf); err != nil {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message(errEMail.String())})
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
	c.Session["lastURL"] = c.Request.URL.String()

	//NOTE: the interceptor assures that the event ID is valid

	if listType != colHasWaitlist && listType != colHasComments {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message("error.undefined")})
	}

	event := models.Event{ID: ID}
	var err error
	if listType == colHasWaitlist {
		err = event.UpdateWaitlist(option, c.Validation)
	} else if listType == colHasComments {
		err = event.UpdateComments(option, c.Validation)
	}

	if err != nil {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message(errDB.String())})
	} else if c.Validation.HasErrors() {
		return c.RenderJSON(
			response{Status: INVALID, Msg: getErrorString(c.Validation.Errors),
				FieldID: listType, Valid: option, ID: ID})
	}

	msg := c.Message("event.waitlist.change.success")
	if listType == colHasComments {
		msg = c.Message("event.has.comments.change.success")
	}

	return c.RenderJSON(
		response{Status: SUCCESS, Msg: msg, FieldID: listType, Valid: option, ID: ID})
}

/*ChangeEnrollmentKey sets an enrollment key.
- Roles: creator and editors of the course of the event */
func (c EditEvent) ChangeEnrollmentKey(ID int, key1, key2, fieldID string) revel.Result {

	c.Log.Debug("change enrollment key", "ID", ID, "key1", key1, "key2", key2, "fieldID", fieldID)
	c.Session["lastURL"] = c.Request.URL.String()

	//NOTE: the interceptor assures that the event ID is valid

	key1 = strings.TrimSpace(key1)
	key2 = strings.TrimSpace(key2)
	if key1 == key2 {

		models.ValidateLength(&key1, "validation.invalid.keys",
			3, 511, c.Validation)

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
	c.Session["lastURL"] = c.Request.URL.String()

	//NOTE: the interceptor assures that the event ID is valid

	event := models.Event{ID: ID}
	_, err := event.Update(nil, "enrollment_key", sql.NullString{
		Valid: false,
	}, nil)

	if err != nil {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message(errDB.String())})
	}

	msg := c.Message("event.key.delete.success")
	return c.RenderJSON(
		response{Status: SUCCESS, Msg: msg, FieldID: "enrollment_key", ID: ID})
}
