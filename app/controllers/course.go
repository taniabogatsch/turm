package controllers

import (
	"turm/app/models"

	"github.com/revel/revel"
)

/*Open an already existing course for enrollment.
- Roles: if public all, else logged in users */
func (c Course) Open(ID int) revel.Result {

	//TODO: implement interceptor

	c.Log.Debug("open course", "ID", ID)

	//NOTE: the interceptor assures that the course ID is valid

	//get the course data
	course := models.Course{ID: ID}
	if err := course.Get(); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	//only set these after the course is loaded
	c.Session["callPath"] = c.Request.URL.String()
	c.Session["currPath"] = c.Request.URL.String()
	c.ViewArgs["tabName"] = c.Message("course")

	c.Log.Debug("loaded course", "course", course)
	return c.Render(course)
}
