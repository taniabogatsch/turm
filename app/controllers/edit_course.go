package controllers

import (
	"database/sql"
	"strings"
	"turm/app/models"
	"turm/app/routes"

	"github.com/revel/revel"
)

//TODO: notify enrolled users if course is updated

/*OpenCourse opens an already existing course for modification, etc.
- Roles: creator and editors of this course. */
func (c EditCourse) OpenCourse(ID int, msg string) revel.Result {

	c.Log.Debug("course management: open course", "ID", ID, "msg", msg)

	//NOTE: the interceptor assures that the course ID is valid

	//get the course data
	course := models.Course{ID: ID}
	if err := course.Get(); err != nil {
		return flashError(
			errDB,
			err,
			routes.Creator.ActiveCourses(),
			c.Controller,
			"",
		)
	}

	//only set these if the course was loaded successfully
	c.Session["callPath"] = c.Request.URL.String()
	c.ViewArgs["tabName"] = c.Message("creator.tab")

	c.Log.Debug("loaded course", "course", course)
	c.Flash.Success(c.Message(msg))
	return c.Render(course)
}

/*ChangeTitle changes the course title.
- Roles: creator and editors of the course */
func (c EditCourse) ChangeTitle(ID int, title string) revel.Result {

	c.Log.Debug("change course title", "ID", ID, "title", title)

	//NOTE: the interceptor assures that the course ID is valid
	title = strings.TrimSpace(title)
	c.Validation.Check(title,
		revel.MinSize{3},
		revel.MaxSize{511},
	).MessageKey("validation.invalid.title")

	if c.Validation.HasErrors() {
		return flashError(
			errValidation,
			nil,
			c.Session["callPath"].(string),
			c.Controller,
			"",
		)
	}

	course := models.Course{ID: ID, Title: title}
	if err := course.Update("title", course.Title); err != nil {
		return flashError(
			errDB,
			err,
			c.Session["callPath"].(string),
			c.Controller,
			"",
		)
	}

	c.Flash.Success(c.Message("course.title.change.success",
		course.Title,
		course.ID,
	))
	return c.Redirect(c.Session["callPath"].(string))
}

/*ChangeSubtitle changes the subtitle.
- Roles: creator and editors of the course */
func (c EditCourse) ChangeSubtitle(ID int, subtitle string) revel.Result {

	c.Log.Debug("change course title", "ID", ID, "subtitle", subtitle)

	//NOTE: the interceptor assures that the course ID is valid
	course := models.Course{ID: ID}
	course.Subtitle = sql.NullString{"", false}

	subtitle = strings.TrimSpace(subtitle)
	if subtitle != "" { //otherwise set subtitle to null
		c.Validation.Check(subtitle,
			revel.MinSize{3},
			revel.MaxSize{511},
		).MessageKey("validation.invalid.subtitle")

		if c.Validation.HasErrors() {
			return flashError(
				errValidation,
				nil,
				c.Session["callPath"].(string),
				c.Controller,
				"",
			)
		}
		course.Subtitle = sql.NullString{subtitle, true}
	}

	if err := course.Update("subtitle", course.Subtitle); err != nil {
		return flashError(
			errDB,
			err,
			c.Session["callPath"].(string),
			c.Controller,
			"",
		)
	}

	if course.Subtitle.Valid {
		c.Flash.Success(c.Message("course.subtitle.change.success",
			course.Subtitle.String,
			course.ID,
		))
	} else {
		c.Flash.Success(c.Message("course.subtitle.delete.success",
			course.ID,
		))
	}
	return c.Redirect(c.Session["callPath"].(string))
}

/*ChangeUnsubscribeEnd changes the unsubscribe end.
- Roles: creator and editors of the course */
func (c EditCourse) ChangeUnsubscribeEnd(ID int, date string, time string) revel.Result {

	c.Log.Debug("change unsubscribe end", "ID", ID, "date", date, "time", time)

	//NOTE: the interceptor assures that the course ID is valid
	course := models.Course{ID: ID}
	course.UnsubscribeEnd = sql.NullString{"", false}

	if date != "" || time != "" {
		c.Validation.Required(date).
			MessageKey("validation.invalid.date")
		c.Validation.Required(time).
			MessageKey("validation.invalid.time")

		course.UnsubscribeEnd = sql.NullString{date + " " + time, true}
		c.Validation.Check(course.UnsubscribeEnd.String,
			models.IsTimestamp{},
		).MessageKey("validation.invalid.timestamp")

	}

	//TODO: if edit, get course, set new unsubscribe end and validate

	if c.Validation.HasErrors() {
		return flashError(
			errValidation,
			nil,
			c.Session["callPath"].(string),
			c.Controller,
			"",
		)
	}

	if err := course.Update("unsubscribeend", course.UnsubscribeEnd); err != nil {
		return flashError(
			errDB,
			err,
			c.Session["callPath"].(string),
			c.Controller,
			"",
		)
	}
	if date != "" || time != "" {
		c.Flash.Success(c.Message("course.unsubscribe.end.change.success",
			course.UnsubscribeEnd.String,
			course.ID,
		))
	} else {
		c.Flash.Success(c.Message("course.unsubscribe.end.delete.success",
			course.ID,
		))
	}
	return c.Redirect(c.Session["callPath"].(string))
}

/*ChangeUserList changes the users on the specified user list of a course.
- Roles: creator and editors of the course */
func (c EditCourse) ChangeUserList(ID int, userID int, listType string) revel.Result {

	c.Log.Debug("change user list", "ID", ID, "userID", userID, "listType", listType)

	//NOTE: the interceptor assures that the course ID is valid
	c.Validation.Required(userID).
		MessageKey("validation.missing.userID")

	if listType != "blacklist" && listType != "whitelist" &&
		listType != "instructor" && listType != "editor" {
		c.Validation.ErrorKey("validation.invalid.params")
	}

	if c.Validation.HasErrors() {
		return flashError(
			errValidation,
			nil,
			c.Session["callPath"].(string),
			c.Controller,
			"",
		)
	}

	entry := models.UserListEntry{UserID: userID, CourseID: ID}
	if err := entry.Insert(listType); err != nil {
		return flashError(
			errDB,
			err,
			c.Session["callPath"].(string),
			c.Controller,
			"",
		)
	}

	c.Flash.Success(c.Message("course."+listType+".change.success", entry.EMail, entry.CourseID))
	return c.Redirect(c.Session["callPath"].(string))
}

/*ChangeViewMatrNr toggles the matriculation number restrictions for an editor/instructor.
- Roles: creators and editors of the course */
//TODO
