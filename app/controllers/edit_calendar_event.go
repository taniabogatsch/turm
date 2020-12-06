package controllers

import (
	"database/sql"
	"strconv"
	"strings"
	"turm/app/models"

	"github.com/revel/revel"
)

/*ChangeText changes the text of the provided column.
- Roles: creator and editors of the course of the calendar event */
func (c EditCalendarEvent) ChangeText(ID int, fieldID, value string) revel.Result {

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

	event := models.CalendarEvent{ID: ID}
	err := event.Update(fieldID, sql.NullString{value, valid})
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

/*Delete calendar event data.
- Roles: creator and editors of the course of the calendar event */
func (c EditCalendarEvent) Delete(ID, courseID int) revel.Result {

	c.Log.Debug("delete calendar event", "ID", ID)

	//NOTE: the interceptor assures that the calendar event ID is valid

	event := models.CalendarEvent{ID: ID}
	if err := event.Delete(); err != nil {
		return flashError(
			errDB, err, "/course/calendarEvents?ID="+strconv.Itoa(courseID),
			c.Controller, "")
	}

	//TODO: send e-mail to all slots if the course is active

	c.Flash.Success(c.Message("event.calendar.delete.success", ID))
	return c.Redirect(Course.CalendarEvents, courseID)
}

/*NewDayTemplate creates a repeatable blueprint of a day.
- Roles: creator and editors of the course of the calendar event */
func (c EditCalendarEvent) NewDayTemplate(ID, courseID int, tmpl models.DayTmpl) revel.Result {

	c.Log.Debug("create a new day template", "ID", ID, "courseID", courseID, "tmpl", tmpl)

	//NOTE: the interceptor assures that the calendar event ID is valid

	tmpl.CalendarEventID = ID
	if err := tmpl.Insert(c.Validation); err != nil {
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

	//NOTE: the interceptor assures that the calendar event ID is valid

	tmpl.CalendarEventID = ID

	data, users, err := tmpl.Update(c.Validation)
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
		mailData := models.EMailData{
			User:        user,
			CourseTitle: data.CourseTitle,
			EventTitle:  data.EventTitle,
			CourseID:    data.CourseID,
		}
		err = sendEMail(c.Controller, &mailData,
			"email.subject.from.slot",
			"manualRemove")
		if err != nil {
			return flashError(
				errEMail, err, "", c.Controller, mailData.User.EMail)
		}
	}

	c.Flash.Success(c.Message("day.tmpl.edit.success", tmpl.ID))
	return c.Redirect(Course.CalendarEvents, courseID)
}

/*ChangeException edits or adds an exception.
- Roles: creator and editors of the course of the calendar event */
func (c EditCalendarEvent) ChangeException(ID, courseID int, exception models.Exception) revel.Result {

	c.Log.Debug("change an exception", "ID", ID, "courseID", courseID, "exception", exception)

	//NOTE: the interceptor assures that the calendar event ID is valid

	exception.CalendarEventID = ID
	data := models.EMailData{}
	users := models.Users{}
	var err error

	//insert
	if exception.ID == 0 {
		data, users, err = exception.Insert(nil, c.Validation)

	} else { //update
		data, users, err = exception.Update(c.Validation)
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
		mailData := models.EMailData{
			User:        user,
			CourseTitle: data.CourseTitle,
			EventTitle:  data.EventTitle,
			CourseID:    data.CourseID,
		}
		err = sendEMail(c.Controller, &mailData,
			"email.subject.from.slot",
			"manualRemove")
		if err != nil {
			return flashError(
				errEMail, err, "", c.Controller, mailData.User.EMail)
		}
	}

	c.Flash.Success(c.Message("exception.change.success", exception.ID))
	return c.Redirect(Course.CalendarEvents, courseID)
}

/*DeleteException deletes an exception.
- Roles: creator and editors of the course of the calendar event */
func (c EditCalendarEvent) DeleteException(ID, courseID int) revel.Result {

	c.Log.Debug("delete an exception", "ID", ID, "courseID", courseID)

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

/*DeleteDayTemplate deletes a day template.
- Roles: creator and editors of the course of the calendar event */
func (c EditCalendarEvent) DeleteDayTemplate(ID, courseID int) revel.Result {

	c.Log.Debug("delete a day template", "ID", ID, "courseID", courseID)

	//NOTE: the interceptor assures that the day template ID is valid

	tmpl := models.DayTmpl{ID: ID}
	if err := tmpl.Delete(); err != nil {
		return flashError(
			errDB, err, "/course/calendarEvents?ID="+strconv.Itoa(courseID),
			c.Controller, "")
	}

	//TODO: return users and write e-mail

	c.Flash.Success(c.Message("day.tmpl.delete.success", ID))
	return c.Redirect(Course.CalendarEvents, courseID)
}
