package controllers

import (
	"turm/app/models"

	"github.com/revel/revel"
)

/*Open an already existing course for enrollment.
- Roles: if public all, else logged in users */
func (c Course) Open(ID int) revel.Result {

	c.Log.Debug("open course", "ID", ID)

	//get user from session
	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		renderQuietError(errTypeConv, err, c.Controller)
		return c.Render()
	}

	//get the course data
	course := models.Course{ID: ID}
	if err := course.Get(nil, false, userID); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	//only set these after the course is loaded
	c.Session["callPath"] = c.Request.URL.String()
	c.Session["currPath"] = c.Request.URL.String()
	c.ViewArgs["tabName"] = c.Message("course")

	//only render content if the course is publicly visible
	if !course.Visible && userID == 0 {
		course = models.Course{
			ID:      course.ID,
			Visible: false,
			Title:   course.Title}
	}

	c.Log.Debug("loaded course", "course", course)
	return c.Render(course)
}
