package controllers

import (
	"database/sql"
	"errors"
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

	msg := c.Message("event." + fieldID + ".delete.success")
	if valid {
		msg = c.Message("event."+fieldID+".change.success", value)
	}

	return c.RenderJSON(
		response{Status: SUCCESS, Msg: msg, FieldID: fieldID, Value: value, ID: ID})
}

/*Delete calendar event data.
- Roles: creator and editors of the course of the calendar event */
func (c EditCalendarEvent) Delete(ID, courseID int) revel.Result {

	c.Log.Debug("delete calendar event", "ID", ID)

	//NOTE: the interceptor assures that the calendar event ID is valid

	//TODO: do not allow the deletion of a calendar event if users are enrolled in it

	event := models.CalendarEvent{ID: ID}
	if err := event.Delete(); err != nil {
		return flashError(
			errDB, err, "/course/events?ID="+strconv.Itoa(courseID),
			c.Controller, "")
	}

	c.Flash.Success(c.Message("event.delete.success", ID))
	return c.Redirect(Course.Events, courseID)
}

/*CreateDayTemplate creates a repeatable blueprint of a day.
- Roles: creator and editors of the course of the calendar event */
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
