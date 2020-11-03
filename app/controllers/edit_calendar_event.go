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
	if err := event.Delete(c.Validation); err != nil {
		return flashError(
			errDB, err, "/course/calendarEvents?ID="+strconv.Itoa(courseID),
			c.Controller, "")
	} else if c.Validation.HasErrors() {
		return flashError(
			errValidation, nil, "/course/calendarEvents?ID="+strconv.Itoa(courseID),
			c.Controller, "")
	}

	c.Flash.Success(c.Message("event.calendar.delete.success", ID))
	return c.Redirect(Course.CalendarEvents, courseID)
}

/*NewDayTemplate creates a repeatable blueprint of a day.
- Roles: creator and editors of the course of the calendar event */
func (c EditCalendarEvent) NewDayTemplate(ID, courseID int, tmpl models.DayTmpl) revel.Result {

	c.Log.Debug("create a new day template", "ID", ID, "courseID", courseID, "tmpl", tmpl)

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
