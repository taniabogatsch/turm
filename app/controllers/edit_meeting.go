package controllers

import (
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
			errValidation,
			nil,
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}

	meeting.ID = ID
	err := meeting.Update()
	if err != nil {
		return flashError(
			errDB,
			err,
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}

	c.Flash.Success(c.Message("meeting.update.success",
		meeting.ID,
	))
	return c.Redirect(c.Session["currPath"])
}

/*Delete a meeting.
- Roles: creator and editors of the course of the meeting */
func (c EditMeeting) Delete(ID int) revel.Result {

	c.Log.Debug("delete meeting", "ID", ID)

	//NOTE: the interceptor assures that the event ID is valid

	meeting := models.Meeting{ID: ID}
	err := meeting.Delete()
	if err != nil {
		return flashError(
			errDB,
			err,
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}

	c.Flash.Success(c.Message("meeting.delete.success",
		ID,
	))
	return c.Redirect(c.Session["currPath"])
}
