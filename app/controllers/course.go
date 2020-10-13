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

/*EditorInstructorList of a course. */
func (c Course) EditorInstructorList(ID int) revel.Result {

	c.Log.Debug("load editors and instructors of course", "ID", ID)

	//TODO: make sure that the course is visible
	//TODO: maybe transaction?
	//TODO: only accessible if user is creator, editor

	editors := models.UserList{}
	if err := editors.Get(nil, &ID, "editors"); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	instructors := models.UserList{}
	if err := instructors.Get(nil, &ID, "instructors"); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	c.Log.Debug("loaded editors and instructors", "editors", editors, "instructors", instructors)
	return c.Render(editors, instructors)
}

/*Whitelist of a course. */
func (c Course) Whitelist(ID int) revel.Result {

	c.Log.Debug("load whitelist of course", "ID", ID)

	//TODO: make sure that the course is visible
	//TODO: only accessible if user is creator, editor

	whitelist := models.UserList{}
	if err := whitelist.Get(nil, &ID, "whitelists"); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	c.Log.Debug("loaded whitelist", "whitelist", whitelist)
	return c.Render(whitelist)
}

/*Blacklist of a course. */
func (c Course) Blacklist(ID int) revel.Result {

	c.Log.Debug("load blacklist of course", "ID", ID)

	//TODO: make sure that the course is visible
	//TODO: only accessible if user is creator, editor

	blacklist := models.UserList{}
	if err := blacklist.Get(nil, &ID, "blacklists"); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	c.Log.Debug("loaded blacklist", "blacklist", blacklist)
	return c.Render(blacklist)
}

/*Path of a course. */
func (c Course) Path(ID int) revel.Result {

	c.Log.Debug("load path of course", "ID", ID)

	//TODO: make sure that the course is visible

	path := models.Groups{}
	if err := path.SelectPath(&ID, nil); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	c.Log.Debug("loaded path", "path", path)
	return c.Render(path)
}

/*Restrictions of a course. */
func (c Course) Restrictions(ID int) revel.Result {

	c.Log.Debug("load restrictions of course", "ID", ID)

	//TODO: make sure that the course is visible

	restrictions := models.Restrictions{}
	if err := restrictions.Get(nil, &ID); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	c.Log.Debug("loaded restrictions", "restrictions", restrictions)
	return c.Render(restrictions)
}

/*Events of a course. */
func (c Course) Events(ID int) revel.Result {

	c.Log.Debug("load events of course", "ID", ID)

	//TODO: make sure that the course is visible
	//TODO: only accessible if user is creator, editor

	events := models.Events{}
	userID := 0
	if err := events.Get(nil, &userID, &ID, true, nil); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	c.Log.Debug("loaded events", "events", events)
	return c.Render(events)
}
