package controllers

import (
	"strings"
	"time"
	"turm/app"
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

	if c.Session["role"] != nil {
		if c.Session["role"].(string) == models.ADMIN.String() {
			course.CanEdit = true
			course.CanManageParticipants = true
			course.IsCreator = true
		} else if int(course.Creator.Int32) == userID {
			course.IsCreator = true
		}
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

	var courses models.CourseList
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

	editors := models.UserList{}
	instructors, err := editors.GetEditorsInstructors(&ID)
	if err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

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

	return c.Render(events)
}

/*CalendarEvent of a course.
- Roles: if public all, else logged in users. */
func (c Course) CalendarEvent(ID, courseID, shift int, monday string) revel.Result {

	c.Log.Debug("load calendar event of course", "ID", ID, "courseID",
		courseID, "shift", shift, "monday", monday)

	loc, err := time.LoadLocation(app.TimeZone)
	if err != nil {
		c.Log.Error("failed to parse location", "loc", app.TimeZone,
			"error", err.Error())
		renderQuietError(errTypeConv, err, c.Controller)
		return c.Render()
	}

	t, err := time.ParseInLocation("2006-01-02T15:04:05-07:00", monday, loc)
	if err != nil {
		c.Log.Error("failed to parse string to time", "monday", monday,
			"loc", loc, "error", err.Error())
		renderQuietError(errTypeConv, err, c.Controller)
		return c.Render()
	}

	t = t.AddDate(0, 0, shift*7)

	event := models.CalendarEvent{ID: ID}
	if err := event.Get(nil, &courseID, t); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	return c.Render(event)
}
