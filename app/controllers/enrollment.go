package controllers

import (
	"fmt"
	"turm/app/models"

	"github.com/revel/revel"
)

/*Enroll a user in an event. */
func (c Enrollment) Enroll(ID int) revel.Result {

	c.Log.Debug("enroll a user in an event", "ID", ID)

	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		return flashError(
			errTypeConv, err, "", c.Controller, "")
	}

	if msg, err := models.EnrollOrUnsubscribe(&userID, &ID, models.ENROLL); err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
	} else if msg != "" {
		c.Validation.ErrorKey(msg)
		return flashError(
			errValidation, nil, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("event.enroll.success"))
	return c.Redirect(c.Session["currPath"])
}

/*Unsubscribe a user from an event. */
func (c Enrollment) Unsubscribe(ID int) revel.Result {

	c.Log.Debug("unsubscribe a user from an event", "ID", ID)

	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		return flashError(
			errTypeConv, err, "", c.Controller, "")
	}

	if msg, err := models.EnrollOrUnsubscribe(&userID, &ID, models.UNSUBSCRIBE); err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
	} else if msg != "" {
		c.Validation.ErrorKey(msg)
		return flashError(
			errValidation, nil, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("event.unsubscribe.success"))
	return c.Redirect(c.Session["currPath"])
}

/*EnrollInCalendarSlot to enroll into a time slot of a day*/
func (c Enrollment) EnrollInCalendarSlot() revel.Result {
	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		return flashError(
			errTypeConv, err, "", c.Controller, "")
	}

	fmt.Println(userID)

	//models.calendar_events.newSlot

	c.Flash.Success(c.Message("event.enroll.success"))
	return c.Redirect(c.Session["currPath"])
}
