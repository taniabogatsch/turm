package controllers

import (
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
func (c Enrollment) EnrollInCalendarSlot(ID int, startTime, endTime, date string) revel.Result {

	/*
		userID, err := getIntFromSession(c.Controller, "userID")
		if err != nil {
			return flashError(
				errTypeConv, err, "", c.Controller, "")
		}

		fmt.Println(userID)

		sT := date + "T" + startTime + "Z"
		sTime, err := time.Parse("2006-01-02T15:04:05Z", sT)
		if err != nil {
			return flashError(errTypeConv, err, "", c.Controller, "")
		}

		eT := date + "T" + endTime + "Z"
		eTime, err := time.Parse("2006-01-02T15:04:05Z", eT)
		if err != nil {
			return flashError(errTypeConv, err, "", c.Controller, "")
		}

		slot := models.Slot{
			StartTimestamp: sTime,
			EndTimestamp:   eTime,
		}

		slot.Validate(c.Validation)

		//check if start time/date is in future and end time/date is afterwards (and valid)

		//models.calendar_events.newSlot
	*/

	c.Flash.Success(c.Message("event.enroll.success"))
	return c.Redirect(c.Session["currPath"])
}
