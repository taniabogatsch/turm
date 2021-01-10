package controllers

import (
	"turm/app"
	"turm/app/models"

	"github.com/revel/revel"
)

//sendEMail sends an e-mail to the specified user
func sendEMail(c *revel.Controller, data *models.EMailData, subjectKey string,
	filename string) (err error) {

	c.Log.Debug("sending EMail", "subjectKey", subjectKey,
		"filename", filename)

	if !data.User.Language.Valid {
		data.User.Language.String = app.DefaultLanguage
	}

	email := app.EMail{
		Recipient: data.User.EMail,
	}

	err = models.GetEMailSubjectBody(
		data,
		&data.User.Language.String,
		subjectKey,
		filename,
		&email,
		c,
	)
	if err != nil {
		return
	}

	app.EMailQueue <- email
	return
}

/*SendEMails to users/editors/instructors after editing the course. */
func sendEMailsEdit(c *revel.Controller, conf *models.EditEMailConfig) (err error) {

	//e-mail data
	data := models.EMailData{
		CourseTitle: conf.CourseTitle,
		EventTitle:  conf.EventTitle,
		CourseID:    conf.CourseID,
		Field:       conf.Field,
	}

	subject := "email.subject.course.edit"
	file := "courseEdit"
	if conf.IsEvent {
		subject = "email.subject.event.edit"
		file = "eventEdit"
	} else if conf.IsCalendarEvent {
		//TODO @Marco
	}

	//send to users
	for _, user := range conf.Users {
		data.User = user
		if err = sendEMail(c, &data, subject, file); err != nil {
			return
		}
	}

	subject = "email.subject.course.edit.manager"
	if conf.IsEvent {
		subject = "email.subject.event.edit.manager"
	}

	//send to editors/instructors
	for _, user := range conf.EditorsInstructors {
		data.User = user
		if err = sendEMail(c, &data, subject, file); err != nil {
			return
		}
	}

	return
}
