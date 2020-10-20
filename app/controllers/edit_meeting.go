package controllers

import (
	"strconv"
	"turm/app/models"

	"github.com/revel/revel"
)

/*Edit meeting data.
- Roles: creator and editors of the course of the meeting */
func (c EditMeeting) Edit(ID int, meeting models.Meeting) revel.Result {

	c.Log.Debug("change meeting", "ID", ID, "meeting", meeting)

	//NOTE: the interceptor assures that the meeting ID is valid

	meeting.Validate(c.Validation)
	if c.Validation.HasErrors() {
		return flashError(
			errValidation, nil, "/course/meetings?ID="+strconv.Itoa(meeting.EventID),
			c.Controller, "")
	}

	meeting.ID = ID
	if err := meeting.Update(); err != nil {
		return flashError(
			errDB, err, "/course/meetings?ID="+strconv.Itoa(meeting.EventID),
			c.Controller, "")
	}

	c.Flash.Success(c.Message("meeting.update.success", meeting.ID))
	return c.Redirect(Course.Meetings, meeting.EventID)
}

/*Delete a meeting.
- Roles: creator and editors of the course of the meeting */
func (c EditMeeting) Delete(ID, eventID int) revel.Result {

	c.Log.Debug("delete meeting", "ID", ID, "eventID", eventID)

	//NOTE: the interceptor assures that the event ID is valid

	meeting := models.Meeting{ID: ID}
	if err := meeting.Delete(); err != nil {
		return flashError(
			errDB, err, "/course/meetings?ID="+strconv.Itoa(eventID),
			c.Controller, "")
	}

	c.Flash.Success(c.Message("meeting.delete.success", ID))
	return c.Redirect(Course.Meetings, eventID)
}
