package controllers

import (
	"turm/app/models"

	"github.com/revel/revel"
)

/*Active renders the active courses page.
- Roles: creator, editors and instructors */
func (c Manage) Active() revel.Result {

	c.Log.Debug("render active courses", "url", c.Request.URL)

	c.Session["callPath"] = c.Request.URL.String()
	c.Session["currPath"] = c.Request.URL.String()
	c.Session["lastURL"] = c.Request.URL.String()

	c.ViewArgs["tab"] = c.Message("creator.tab")

	//get the user
	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		renderQuietError(errTypeConv, err, c.Controller)
		return c.Render()
	}

	creator := models.CourseList{}
	editor, instructor, err := creator.Get(true, false, userID, c.Session["role"].(string))
	if err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	return c.Render(creator, editor, instructor)
}

/*Drafts renders the drafts page.
- Roles: creator and editors */
func (c Manage) Drafts() revel.Result {

	c.Log.Debug("render drafts page", "url", c.Request.URL)

	c.Session["callPath"] = c.Request.URL.String()
	c.Session["currPath"] = c.Request.URL.String()
	c.Session["lastURL"] = c.Request.URL.String()

	c.ViewArgs["tab"] = c.Message("creator.tab")

	//get the user
	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		renderQuietError(errTypeConv, err, c.Controller)
		return c.Render()
	}

	creator := models.CourseList{}
	editor, _, err := creator.Get(false, false, userID, c.Session["role"].(string))
	if err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	return c.Render(creator, editor)
}

/*Expired renders the expired courses page.
- Roles: creator, editors and instructors */
func (c Manage) Expired() revel.Result {

	c.Log.Debug("render expired courses", "url", c.Request.URL)

	c.Session["callPath"] = c.Request.URL.String()
	c.Session["currPath"] = c.Request.URL.String()
	c.Session["lastURL"] = c.Request.URL.String()

	c.ViewArgs["tab"] = c.Message("creator.tab")

	//get the user
	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		renderQuietError(errTypeConv, err, c.Controller)
		return c.Render()
	}

	creator := models.CourseList{}
	editor, instructor, err := creator.Get(true, true, userID, c.Session["role"].(string))
	if err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	return c.Render(creator, editor, instructor)
}
