package controllers

import (
	"turm/app/models"

	"github.com/revel/revel"
)

/*Open a course for user management. */
func (c Participants) Open(ID int) revel.Result {

	c.Log.Debug("open course for user management", "ID", ID)

	//TODO: the interceptor assures that the course ID is valid

	//get the course data
	participants := models.Participants{ID: ID}
	if err := participants.Get(); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	//only set these after the course is loaded
	c.Session["callPath"] = c.Request.URL.String()
	c.Session["currPath"] = c.Request.URL.String()
	c.ViewArgs["tabName"] = c.Message("creator.tab")

	return c.Render(participants)
}

/*Open*/
