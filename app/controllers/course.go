package controllers

import (
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

	//only set these after the course is loaded - TODO: why?
	c.Session["callPath"] = c.Request.URL.String()
	c.Session["currPath"] = c.Request.URL.String()
	c.Session["lastURL"] = c.Request.URL.String()
	c.ViewArgs["tab"] = c.Message("course")

	return c.Render(course)
}

/*Search for a specific course.
Roles: all (except not activated users). */
func (c Course) Search(value string) revel.Result {

	c.Log.Debug("search courses", "value", value)
	c.Session["lastURL"] = c.Request.URL.String()

	models.ValidateLength(&value, "validation.invalid.searchValue",
		1, 127, c.Validation)

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
	c.Session["lastURL"] = c.Request.URL.String()

	editors := models.UserList{}
	instructors, err := editors.GetEditorsInstructors(&ID)
	if err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	return c.Render(editors, instructors)
}

/*Allowlist of a course.
- Roles: creator and editors of this course. */
func (c Course) Allowlist(ID int) revel.Result {

	c.Log.Debug("load allowlist of course", "ID", ID)
	c.Session["lastURL"] = c.Request.URL.String()

	allowlist := models.UserList{}
	if err := allowlist.Get(nil, &ID, models.TableAllowlists); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	return c.Render(allowlist)
}

/*Blocklist of a course.
- Roles: creator and editors of this course. */
func (c Course) Blocklist(ID int) revel.Result {

	c.Log.Debug("load blocklist of course", "ID", ID)
	c.Session["lastURL"] = c.Request.URL.String()

	blocklist := models.UserList{}
	if err := blocklist.Get(nil, &ID, models.TableBlocklists); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	return c.Render(blocklist)
}

/*Path of a course.
- Roles: if public all, else logged in users. */
func (c Course) Path(ID int) revel.Result {

	c.Log.Debug("load path of course", "ID", ID)
	c.Session["lastURL"] = c.Request.URL.String()

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
	c.Session["lastURL"] = c.Request.URL.String()

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
	c.Session["lastURL"] = c.Request.URL.String()

	events := models.Events{}
	userID := 0 //edit page, so the user is never allowed to enroll
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
	c.Session["lastURL"] = c.Request.URL.String()

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
	c.Session["lastURL"] = c.Request.URL.String()

	//get the last (current) monday
	now := time.Now()
	weekday := time.Now().Weekday()
	monday := now.AddDate(0, 0, -1*(int(weekday)-1))

	userID := 0 //edit page, so the user is never allowed to enroll
	events := models.CalendarEvents{}
	if err := events.Get(nil, &ID, monday, userID); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	return c.Render(events)
}

/*CalendarEvent of a course. Loads a specific calender event as defined by monday.
- Roles: if public all, else logged in users. */
func (c Course) CalendarEvent(ID, courseID, shift int, monday string, day int) revel.Result {

	c.Log.Debug("load calendar event of course", "ID", ID, "courseID",
		courseID, "shift", shift, "monday", monday, "day", day)
	c.Session["lastURL"] = c.Request.URL.String()

	//get user from session
	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		renderQuietError(errTypeConv, err, c.Controller)
		return c.Render()
	}

	loc, err := time.LoadLocation(app.TimeZone)
	if err != nil {
		c.Log.Error("failed to parse location", "loc", app.TimeZone,
			"error", err.Error())
		renderQuietError(errTypeConv, err, c.Controller)
		return c.Render()
	}

	//try this time format
	t, err := time.ParseInLocation("2006-01-02T15:04:05-07:00", monday, loc)
	if err != nil {

		//test another time format
		t, err = time.ParseInLocation("2006-01-02T15:04:05.999999999Z", monday, loc)

		//no matching format
		if err != nil {
			c.Log.Error("failed to parse string to time", "monday", monday,
				"loc", loc, "error", err.Error())
			renderQuietError(errTypeConv, err, c.Controller)
			return c.Render()
		}

	}

	t = t.AddDate(0, 0, shift*7)

	event := models.CalendarEvent{ID: ID}
	if err := event.Get(nil, &courseID, t, userID); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	if shift == -1 {
		day = 6
	} else if shift == 1 {
		day = 0
	}

	return c.Render(event, day)
}
