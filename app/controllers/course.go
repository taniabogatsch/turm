package controllers

import (
	"strings"
	"time"
	"turm/app/models"

	"github.com/revel/revel"
)

/*Open an already existing course for enrollment.
- Roles: if public all, else logged in users. */
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

	//only render content if the course is publicly visible
	if !course.Visible && userID == 0 {
		course = models.Course{
			ID:      course.ID,
			Visible: false,
			Title:   course.Title}
	}

	//only set these after the course is loaded
	c.Session["callPath"] = c.Request.URL.String()
	c.Session["currPath"] = c.Request.URL.String()
	c.ViewArgs["tabName"] = c.Message("course")

	c.Log.Debug("loaded course", "course", course)
	return c.Render(course)
}

/*Search for a specific course.
Roles: all (except not activated users). */
func (c Course) Search(value string) revel.Result {

	c.Log.Debug("search courses", "value", value)

	value = strings.TrimSpace(value)
	c.Validation.Check(value,
		revel.MinSize{1},
		revel.MaxSize{127},
	).MessageKey("validation.invalid.searchValue")

	if c.Validation.HasErrors() {
		c.Validation.Keep()
		return c.Render()
	}

	var courses models.Courses
	if err := courses.Search(value); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	return c.Render(courses)
}

/*EditorInstructorList of a course.
- Roles: if public all, else logged in users. */
func (c Course) EditorInstructorList(ID int) revel.Result {

	c.Log.Debug("load editors and instructors of course", "ID", ID)

	//TODO: maybe transaction?

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

/*Whitelist of a course.
- Roles: creator and editors of this course. */
func (c Course) Whitelist(ID int) revel.Result {

	c.Log.Debug("load whitelist of course", "ID", ID)

	whitelist := models.UserList{}
	if err := whitelist.Get(nil, &ID, "whitelists"); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	c.Log.Debug("loaded whitelist", "whitelist", whitelist)
	return c.Render(whitelist)
}

/*Blacklist of a course.
- Roles: creator and editors of this course. */
func (c Course) Blacklist(ID int) revel.Result {

	c.Log.Debug("load blacklist of course", "ID", ID)

	blacklist := models.UserList{}
	if err := blacklist.Get(nil, &ID, "blacklists"); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	c.Log.Debug("loaded blacklist", "blacklist", blacklist)
	return c.Render(blacklist)
}

/*Path of a course.
- Roles: if public all, else logged in users. */
func (c Course) Path(ID int) revel.Result {

	c.Log.Debug("load path of course", "ID", ID)

	path := models.Groups{}
	if err := path.SelectPath(&ID, nil); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	c.Log.Debug("loaded path", "path", path)
	return c.Render(path)
}

/*Restrictions of a course.
- Roles: if public all, else logged in users. */
func (c Course) Restrictions(ID int) revel.Result {

	c.Log.Debug("load restrictions of course", "ID", ID)

	restrictions := models.Restrictions{}
	if err := restrictions.Get(nil, &ID); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	c.Log.Debug("loaded restrictions", "restrictions", restrictions)
	return c.Render(restrictions)
}

/*Events of a course.
- Roles: if public all, else logged in users. */
func (c Course) Events(ID int) revel.Result {

	c.Log.Debug("load events of course", "ID", ID)

	events := models.Events{}
	userID := 0
	if err := events.Get(nil, &userID, &ID, true, nil); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	manage := true
	c.Log.Debug("loaded events", "events", events)
	return c.Render(events, manage)
}

/*Meetings of an event.
- Roles: if public all, else logged in users. */
func (c Course) Meetings(ID int) revel.Result {

	c.Log.Debug("load meetings of an event", "ID", ID)

	meetings := models.Meetings{}
	if err := meetings.Get(nil, &ID); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	c.Log.Debug("loaded meetings", "meetings", meetings)
	return c.Render(meetings, ID)
}

/*CalendarEvents of a course.
- Roles: if public all, else logged in users. */
func (c Course) CalendarEvents(ID int) revel.Result {

	c.Log.Debug("load calendar events of course", "ID", ID)

	//get the last (current) monday
	now := time.Now()
	weekday := time.Now().Weekday()
	monday := now.AddDate(0, 0, -1*(int(weekday)-1))

	events := models.CalendarEvents{}
	if err := events.Get(nil, &ID, monday); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	c.Log.Debug("loaded calendar events", "calendar events", events)
	return c.Render(events)
}
