package models

import (
	"path/filepath"
	"turm/app"

	"github.com/revel/revel"
)

/*EMailData holds all data that is rendered in the different e-mail templates. */
type EMailData struct {
	User User
	URL  string
}

/*GetEMailSubjectBody assigns the template content to the e-mail body and sets the e-mail subject. */
func GetEMailSubjectBody(data *EMailData, language *string, subjectKey string,
	filename string, email *app.EMail, c *revel.Controller) (err error) {

	data.URL = app.URL
	c.ViewArgs["data"] = data //set the data for parsing the e-mail body

	cLanguage := c.Session["currentLocale"].(string)
	c.ViewArgs["currentLocale"] = *language //set the preferred language for template parsing
	c.Request.Locale = *language

	email.Subject = c.Message(subjectKey) //set the e-mail subject
	email.ReplyTo = c.Message("email.no.reply", app.ServiceEMail)

	filepath := filepath.Join("emails", filename+"_"+*language+".html")

	//parse template / e-mail body
	buf, err := revel.TemplateOutputArgs(filepath, c.ViewArgs)
	if err != nil {
		modelsLog.Error("failed to parse e-mail template", "filepath", filepath,
			"viewArgs", c.ViewArgs, "error", err.Error())
		return
	}
	email.Body = string(buf)

	c.ViewArgs["currentLocale"] = cLanguage //reset to original language
	c.Request.Locale = cLanguage
	return
}
