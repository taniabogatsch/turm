package controllers

import (
	"turm/app"
	"turm/app/models"

	"github.com/revel/revel"
)

//sendEMail sends an e-mail to the specified user
func sendEMail(c *revel.Controller, user *models.User, subjectKey string, filename string) (err error) {

	c.Log.Debug("sending EMail", "user", user, "subjectKey", subjectKey,
		"filename", filename)

	data := models.EMailData{User: *user}

	if !user.Language.Valid {
		user.Language.String = app.DefaultLanguage
	}

	email := app.EMail{
		Recipient: user.EMail,
	}

	err = models.GetEMailSubjectBody(
		&data,
		&user.Language.String,
		subjectKey,
		filename,
		&email,
		c,
	)
	if err != nil {
		return
	}

	c.Log.Debug("assembled e-mail", "email", email)

	app.EMailQueue <- email
	return
}
