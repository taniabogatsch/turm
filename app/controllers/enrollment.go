package controllers

import (
	"fmt"
	"time"
	"turm/app/models"

	"github.com/revel/revel"
)

/*Enroll a user in an event. */
func (c Enrollment) Enroll(ID int, key string) revel.Result {

	c.Log.Debug("enroll a user in an event", "ID", ID, "key", key)

	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		return flashError(
			errTypeConv, err, "", c.Controller, "")
	}

	//enroll user
	data, waitList, _, msg, err := models.EnrollOrUnsubscribe(&userID, &ID, models.ENROLL, key)
	if err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
	} else if msg != "" {
		c.Validation.ErrorKey(msg)
		return flashError(
			errValidation, nil, "", c.Controller, "")
	}

	//send e-mail to the user who enrolled
	if waitList {
		err = sendEMail(c.Controller, &data,
			"email.subject.wait.list",
			"waitlist")
	} else {
		err = sendEMail(c.Controller, &data,
			"email.subject.enroll",
			"enroll")
	}
	if err != nil {
		return flashError(
			errEMail, err, "", c.Controller, data.User.EMail)
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

	//unsubscribe user
	data, waitList, users, msg, err := models.EnrollOrUnsubscribe(&userID, &ID, models.UNSUBSCRIBE, "")
	if err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
	} else if msg != "" {
		c.Validation.ErrorKey(msg)
		return flashError(
			errValidation, nil, "", c.Controller, "")
	}

	//send e-mail to the user who unsubscribed
	if waitList {
		err = sendEMail(c.Controller, &data,
			"email.subject.unsub.wait.list",
			"unsubWaitlist")
	} else {
		err = sendEMail(c.Controller, &data,
			"email.subject.unsubscribe",
			"unsubscribe")
	}
	if err != nil {
		return flashError(
			errEMail, err, "", c.Controller, data.User.EMail)
	}

	//send e-mail to each auto enrolled user
	if len(users) != 0 {
		for _, user := range users {
			mailData := models.EMailData{
				User:        user,
				CourseTitle: data.CourseTitle,
				EventTitle:  data.EventTitle,
				CourseID:    data.CourseID,
			}
			err = sendEMail(c.Controller, &mailData,
				"email.subject.from.wait.list",
				"fromWaitlist")
			if err != nil {
				return flashError(
					errEMail, err, "", c.Controller, mailData.User.EMail)
			}
		}
	}

	c.Flash.Success(c.Message("event.unsubscribe.success"))
	return c.Redirect(c.Session["currPath"])
}

/*EnrollInSlot to enroll into a time slot of a day in a calendar event. */
func (c Enrollment) EnrollInSlot(ID int, startTime, endTime, date string) revel.Result {

	c.Log.Debug("enroll a user in an calendar event", "ID", ID, "startTime",
		startTime, "endTime", endTime, "date", date)

	//get user
	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		return flashError(errTypeConv, err, "", c.Controller, "")
	}

	//set start and end time
	startT := date + "T" + startTime + "Z"
	start, err := time.Parse("2006-01-02T15:04:05Z", startT)
	if err != nil {
		return flashError(errTypeConv, err, "", c.Controller, "")
	}
	endT := date + "T" + endTime + "Z"
	end, err := time.Parse("2006-01-02T15:04:05Z", endT)
	if err != nil {
		return flashError(errTypeConv, err, "", c.Controller, "")
	}

	slot := models.Slot{
		UserID: userID,
		Start:  start,
		End:    end,
	}

	//enroll user
	data, err := slot.Insert(c.Validation)
	if err != nil {
		return flashError(errDB, err, "", c.Controller, "")
	} else if c.Validation.HasErrors() {
		return flashError(errValidation, nil, "", c.Controller, "")
	}

	fmt.Println(data)
	//TODO: send e-mail to the user who enrolled
	/*
		err = sendEMail(c.Controller, &data,
			"email.subject.enroll.slot",
			"enrollToSlot")
		if err != nil {
			return flashError(
				errEMail, err, "", c.Controller, data.User.EMail)
		}
	*/

	c.Flash.Success(c.Message("event.enroll.success"))
	return c.Redirect(c.Session["currPath"])
}
