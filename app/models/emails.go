package models

import (
	"turm/app"

	"github.com/revel/revel"
)

/*EMailData holds all data that is rendered in the different e-mail templates. */
type EMailData struct {
	User User
	URL  string
}

/*GetEMailBody assigns the template content to the e-mail body an e-mail body. */
func GetEMailBody(data *EMailData, filepath string, body *string, c *revel.Controller) (err error) {

	data.URL = app.URL
	c.ViewArgs["data"] = data

	//parse template
	buf, err := revel.TemplateOutputArgs(filepath, c.ViewArgs)
	if err != nil {
		revel.AppLog.Error("failed to parse e-mail template", "filepath", filepath,
			"viewArgs", c.ViewArgs, "error", err.Error())
		return
	}

	*body = string(buf)
	return
}
