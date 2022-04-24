package controllers

import (
	"database/sql"
	"strconv"
	"strings"
	"turm/app/models"

	"github.com/revel/revel"
)

/*ChangeText of the provided column of a calendar event.
- Roles: creator and editors of the course of the calendar event */
func (c EditCalendarEvent) ChangeText(ID int, fieldID, value string) revel.Result {

	c.Log.Debug("change text value", "ID", ID, "fieldID", fieldID, "value", value)
	c.Session["lastURL"] = c.Request.URL.String()

	value = strings.TrimSpace(value)
	valid := (value != "")

	//NOTE: the interceptor assures that the event ID is valid

	if value != "" || fieldID == models.ColTitle {

		models.ValidateLength(&value, "validation.invalid.text.short",
			3, 255, c.Validation)

		if c.Validation.HasErrors() {
			return c.RenderJSON(
				response{Status: INVALID, Msg: getErrorString(c.Validation.Errors)})
		}
	}

	if fieldID != models.ColTitle && fieldID != models.ColAnnotation {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message("error.undefined")})
	}

	event := models.CalendarEvent{ID: ID}
	err := event.Update(fieldID, sql.NullString{
		String: value,
		Valid:  valid,
	})
	if err != nil {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message(errDB.String())})
	}

	msg := c.Message("event.calendar." + fieldID + ".delete.success")
	if valid {
		msg = c.Message("event.calendar."+fieldID+".change.success", value)
	}

	//TODO: notify people (according to conf) about changes

	return c.RenderJSON(
		response{Status: SUCCESS, Msg: msg, FieldID: "calendar_" + fieldID,
			Value: value, ID: ID})
}

/*Delete calendar event.
- Roles: creator and editors of the course of the calendar event */
func (c EditCalendarEvent) Delete(ID, courseID int) revel.Result {

	c.Log.Debug("delete calendar event", "ID", ID, "courseID", courseID)
	c.Session["lastURL"] = c.Request.URL.String()

	//NOTE: the interceptor assures that the calendar event ID is valid

	event := models.CalendarEvent{ID: ID}
	users, err := event.Delete(c.Validation)
	if err != nil {
		return flashError(
			errDB, err, "/course/calendarEvents?ID="+strconv.Itoa(courseID),
			c.Controller, "")
	} else if c.Validation.HasErrors() {
		return flashError(
			errValidation, err, "/course/calendarEvents?ID="+strconv.Itoa(courseID),
			c.Controller, "")
	}

	//send e-mail to all upcoming slots if the course is active
	for _, user := range users {

		err = sendEMail(c.Controller, &user,
			"email.subject.from.slot",
			"manualRemove")
		if err != nil {
			return flashError(errEMail, err, "", c.Controller, user.User.EMail)
		}
	}

	c.Flash.Success(c.Message("event.calendar.delete.success", ID))
	return c.Redirect(Course.CalendarEvents, courseID)
}

/*Duplicate calendar event.
- Roles: creator and editors of the course of the event */
func (c EditCalendarEvent) Duplicate(ID, courseID int) revel.Result {

	c.Log.Debug("duplicate calendar event", "ID", ID, "courseID", courseID)
	c.Session["lastURL"] = c.Request.URL.String()

	//NOTE: the interceptor assures that the event ID is valid

	event := models.CalendarEvent{ID: ID, CourseID: courseID}
	if err := event.Duplicate(nil); err != nil {
		return flashError(
			errDB, err, "/course/calendarEvents?ID="+strconv.Itoa(courseID),
			c.Controller, "")
	}

	c.Flash.Success(c.Message("event.duplicate.success", ID))
	return c.Redirect(Course.CalendarEvents, courseID)
}

/*NewDayTemplate creates a repeatable blueprint of a day.
- Roles: creator and editors of the course of the calendar event */
func (c EditCalendarEvent) NewDayTemplate(ID, courseID int, tmpl models.DayTmpl) revel.Result {

	c.Log.Debug("create a new day template", "ID", ID, "courseID", courseID, "tmpl", tmpl)
	c.Session["lastURL"] = c.Request.URL.String()

	//NOTE: the interceptor assures that the calendar event ID is valid

	tmpl.CalendarEventID = ID
	if err := tmpl.Insert(nil, c.Validation); err != nil {
		return flashError(
			errDB, err, "/course/calendarEvents?ID="+strconv.Itoa(courseID),
			c.Controller, "")
	} else if c.Validation.HasErrors() {
		return flashError(
			errValidation, err, "/course/calendarEvents?ID="+strconv.Itoa(courseID),
			c.Controller, "")
	}

	c.Flash.Success(c.Message("day.tmpl.new.success", tmpl.ID))
	return c.Redirect(Course.CalendarEvents, courseID)
}

/*EditDayTemplate edits a repeatable blueprint of a day.
- Roles: creator and editors of the course of the calendar event */
func (c EditCalendarEvent) EditDayTemplate(ID, courseID int, tmpl models.DayTmpl) revel.Result {

	c.Log.Debug("edit a day template", "ID", ID, "courseID", courseID, "tmpl", tmpl)
	c.Session["lastURL"] = c.Request.URL.String()

	//NOTE: the interceptor assures that the calendar event ID is valid

	tmpl.CalendarEventID = ID

	users, err := tmpl.Update(c.Validation)
	if err != nil {
		return flashError(
			errDB, err, "/course/calendarEvents?ID="+strconv.Itoa(courseID),
			c.Controller, "")
	} else if c.Validation.HasErrors() {
		return flashError(
			errValidation, err, "/course/calendarEvents?ID="+strconv.Itoa(courseID),
			c.Controller, "")
	}

	//TODO: when updating, validate that the tmpl ID fits the calendar event ID

	//send e-mail to each user that got removed from its slot
	for _, user := range users {

		err = sendEMail(c.Controller, &user,
			"email.subject.from.slot",
			"manualRemove")
		if err != nil {
			return flashError(errEMail, err, "", c.Controller, user.User.EMail)
		}
	}

	c.Flash.Success(c.Message("day.tmpl.edit.success", tmpl.ID))
	return c.Redirect(Course.CalendarEvents, courseID)
}

/*ChangeException edits or adds an exception. Exceptions block slots of a day template.
- Roles: creator and editors of the course of the calendar event */
func (c EditCalendarEvent) ChangeException(ID, courseID int, exception models.Exception) revel.Result {

	c.Log.Debug("change an exception", "ID", ID, "courseID", courseID, "exception", exception)
	c.Session["lastURL"] = c.Request.URL.String()

	//NOTE: the interceptor assures that the calendar event ID is valid

	exception.CalendarEventID = ID
	var users []models.EMailData
	var err error

	//insert
	if exception.ID == 0 {
		users, err = exception.Insert(nil, c.Validation)

	} else { //update
		users, err = exception.Update(c.Validation)
		//TODO: when updating, validate that the exception ID fits the calendar event ID
	}

	if err != nil {
		return flashError(
			errDB, err, "/course/calendarEvents?ID="+strconv.Itoa(courseID),
			c.Controller, "")
	} else if c.Validation.HasErrors() {
		return flashError(
			errValidation, err, "/course/calendarEvents?ID="+strconv.Itoa(courseID),
			c.Controller, "")
	}

	//send e-mail to each user that got removed from its slot
	for _, user := range users {

		err = sendEMail(c.Controller, &user,
			"email.subject.from.slot",
			"manualRemove")
		if err != nil {
			return flashError(errEMail, err, "", c.Controller, user.User.EMail)
		}
	}

	c.Flash.Success(c.Message("exception.change.success", exception.ID))
	return c.Redirect(Course.CalendarEvents, courseID)
}

/*DeleteException of a calendar event.
- Roles: creator and editors of the course of the calendar event */
func (c EditCalendarEvent) DeleteException(ID, courseID int) revel.Result {

	c.Log.Debug("delete an exception", "ID", ID, "courseID", courseID)
	c.Session["lastURL"] = c.Request.URL.String()

	//NOTE: the interceptor assures that the day template ID is valid

	exception := models.Exception{ID: ID}
	if err := exception.Delete(nil); err != nil {
		return flashError(
			errDB, err, "/course/calendarEvents?ID="+strconv.Itoa(courseID),
			c.Controller, "")
	}

	c.Flash.Success(c.Message("exception.delete.success", ID))
	return c.Redirect(Course.CalendarEvents, courseID)
}

/*DeleteDayTemplate of a calendar event.
- Roles: creator and editors of the course of the calendar event */
func (c EditCalendarEvent) DeleteDayTemplate(ID, courseID int) revel.Result {

	c.Log.Debug("delete a day template", "ID", ID, "courseID", courseID)
	c.Session["lastURL"] = c.Request.URL.String()

	//NOTE: the interceptor assures that the day template ID is valid

	tmpl := models.DayTmpl{ID: ID}
	users, err := tmpl.Delete()
	if err != nil {
		return flashError(
			errDB, err, "/course/calendarEvents?ID="+strconv.Itoa(courseID),
			c.Controller, "")
	}

	//send e-mail to each user that got removed from its slot
	for _, user := range users {

		err = sendEMail(c.Controller, &user,
			"email.subject.from.slot",
			"manualRemove")
		if err != nil {
			return flashError(errEMail, err, "", c.Controller, user.User.EMail)
		}
	}

	c.Flash.Success(c.Message("day.tmpl.delete.success", ID))
	return c.Redirect(Course.CalendarEvents, courseID)
}
