package controllers

import (
	"strconv"
	"strings"
	"time"
	"turm/app"
	"turm/app/models"

	"github.com/revel/revel"
)

/*Enroll a user in an event.
- Roles: logged in and activated users */
func (c Enrollment) Enroll(ID int, key string) revel.Result {

	c.Log.Debug("enroll a user in an event", "ID", ID, "key", key)

	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		return flashError(
			errTypeConv, err, "", c.Controller, "")
	}

	//enroll user
	enrolled := models.Enrolled{EventID: ID, UserID: userID}
	data, waitList, _, msg, err := enrolled.EnrollOrUnsubscribe(models.ENROLL, key)
	if err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
	} else if msg != "" {
		c.Validation.ErrorKey(msg)
		return flashError(
			errValidation, nil, "", c.Controller, "")
	}

	//send e-mail to the user
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

/*Unsubscribe a user from an event.
- Roles: logged in and activated users */
func (c Enrollment) Unsubscribe(ID int) revel.Result {

	c.Log.Debug("unsubscribe a user from an event", "ID", ID)

	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		return flashError(
			errTypeConv, err, "", c.Controller, "")
	}

	//unsubscribe user
	enrolled := models.Enrolled{EventID: ID, UserID: userID}
	data, waitList, users, msg, err := enrolled.EnrollOrUnsubscribe(models.UNSUBSCRIBE, "")
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
			return flashError(errEMail, err, "", c.Controller, mailData.User.EMail)
		}
	}

	c.Flash.Success(c.Message("event.unsubscribe.success"))
	return c.Redirect(c.Session["currPath"])
}

/*EnrollInSlot to enroll into a time slot of a day in a calendar event.
- Roles: logged in and activated users */
func (c Enrollment) EnrollInSlot(ID, year int, startTime, endTime, date string) revel.Result {

	c.Log.Debug("enroll a user in an calendar event", "ID", ID, "year", year, "startTime",
		startTime, "endTime", endTime, "date", date)

	//get user
	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		return flashError(errTypeConv, err, "", c.Controller, "")
	}

	location, err := time.LoadLocation(app.TimeZone)
	if err != nil {
		return flashError(errTypeConv, err, "", c.Controller, "")
	}

	splitDate := strings.Split(date, ".")
	month, err := strconv.Atoi(splitDate[1])
	if err != nil {
		return flashError(errTypeConv, err, "", c.Controller, "")
	}
	day, err := strconv.Atoi(splitDate[0])
	if err != nil {
		return flashError(errTypeConv, err, "", c.Controller, "")
	}

	//set start and end time
	splitTime := strings.Split(startTime, ":")
	hour, err := strconv.Atoi(splitTime[0])
	if err != nil {
		return flashError(errTypeConv, err, "", c.Controller, "")
	}
	min, err := strconv.Atoi(splitTime[1])
	if err != nil {
		return flashError(errTypeConv, err, "", c.Controller, "")
	}

	start := time.Date(year, time.Month(month), day, hour, min, 0, 0, location)

	splitTime = strings.Split(endTime, ":")
	hour, err = strconv.Atoi(splitTime[0])
	if err != nil {
		return flashError(errTypeConv, err, "", c.Controller, "")
	}
	min, err = strconv.Atoi(splitTime[1])
	if err != nil {
		return flashError(errTypeConv, err, "", c.Controller, "")
	}

	end := time.Date(year, time.Month(month), day, hour, min, 0, 0, location)

	slot := models.Slot{
		UserID: userID,
		Start:  start,
		End:    end,
	}

	//enroll user
	data, err := slot.Insert(c.Validation, ID)
	if err != nil {
		return flashError(errDB, err, "", c.Controller, "")
	} else if c.Validation.HasErrors() {
		return flashError(errValidation, nil, "", c.Controller, "")
	}

	//send e-mail to the user who enrolled
	err = sendEMail(c.Controller, &data,
		"email.subject.enroll.slot",
		"enrollToSlot")
	if err != nil {
		return flashError(
			errEMail, err, "", c.Controller, data.User.EMail)
	}

	c.Flash.Success(c.Message("event.enroll.success"))
	return c.Redirect(c.Session["currPath"])
}
