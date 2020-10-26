package controllers

import (
	"turm/app"
	"turm/app/models"

	"github.com/revel/revel"
)

//sendEMail sends an e-mail to the specified user
func sendEMail(c *revel.Controller, data *models.EMailData, subjectKey string,
	filename string) (err error) {

	c.Log.Debug("sending EMail", "data", *data, "subjectKey", subjectKey,
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

	c.Log.Debug("assembled e-mail", "email", email)

	app.EMailQueue <- email
	return
}
