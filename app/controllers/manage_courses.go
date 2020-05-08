package controllers

import (
	"strconv"
	"turm/app/models"

	"github.com/revel/revel"
)

/*Active renders the active courses page.
- Roles: creator, editors and instructors */
func (c ManageCourses) Active() revel.Result {

	c.Log.Debug("render active courses", "url", c.Request.URL)
	c.Session["callPath"] = c.Request.URL.String()
	c.Session["currPath"] = c.Request.URL.String()
	c.ViewArgs["tabName"] = c.Message("creator.tab")

	return c.Render()
}

/*GetActive renders all active courses of the current user.
- Roles: creator, editors and instructors */
func (c ManageCourses) GetActive() revel.Result {

	c.Log.Debug("render active courses")

	creator, editor, instructor, err := c.getCourseLists(true, false)
	if err != nil {
		return renderError(
			err,
			c.Controller,
		)
	}
	return c.Render(creator, editor, instructor)
}

/*Drafts renders the drafts page.
- Roles: creator and editors */
func (c ManageCourses) Drafts() revel.Result {

	c.Log.Debug("render drafts page", "url", c.Request.URL)
	c.Session["callPath"] = c.Request.URL.String()
	c.Session["currPath"] = c.Request.URL.String()
	c.ViewArgs["tabName"] = c.Message("creator.tab")

	return c.Render()
}

/*GetDrafts renders all inactive courses of the current user.
- Roles: creator and editors */
func (c ManageCourses) GetDrafts() revel.Result {

	c.Log.Debug("render drafts")

	creator, editor, _, err := c.getCourseLists(false, false)
	if err != nil {
		return renderError(
			err,
			c.Controller,
		)
	}
	return c.Render(creator, editor)
}

/*Expired renders the expired courses page.
- Roles: creator, editors and instructors */
func (c ManageCourses) Expired() revel.Result {

	c.Log.Debug("render expired courses", "url", c.Request.URL)
	c.Session["callPath"] = c.Request.URL.String()
	c.Session["currPath"] = c.Request.URL.String()
	c.ViewArgs["tabName"] = c.Message("creator.tab")

	return c.Render()
}

/*GetExpired renders all expired courses of the current user.
- Roles: creator, editors and instructors */
func (c ManageCourses) GetExpired() revel.Result {

	c.Log.Debug("render expired courses")

	creator, editor, instructor, err := c.getCourseLists(true, true)
	if err != nil {
		return renderError(
			err,
			c.Controller,
		)
	}
	return c.Render(creator, editor, instructor)
}

//getCourseLists returns the specified course lists
func (c ManageCourses) getCourseLists(active, expired bool) (
	creator, editor, instructor models.CourseList, err error) {

	//if the user is an admin, render all active courses
	if c.Session["role"] == models.ADMIN.String() {
		userID := 0
		err = creator.GetByUserID(&userID, "admin", active, expired)
		return
	}

	//get the user
	userID, err := strconv.Atoi(c.Session["userID"].(string))
	if err != nil {
		c.Log.Error("failed to parse userID from session",
			"session", c.Session, "error", err.Error())
		return
	}

	if err = creator.GetByUserID(&userID, "creator", active, expired); err != nil {
		return
	}
	if err = editor.GetByUserID(&userID, "editor", active, expired); err != nil {
		return
	}
	if active { //instructors are not part of drafts
		err = instructor.GetByUserID(&userID, "instructor", active, expired)
	}
	return
}
