package controllers

import (
	"database/sql"
	"strings"
	"turm/app/models"

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
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	//only set these after the course is loaded
	c.Session["callPath"] = c.Request.URL.String()
	c.Session["currPath"] = c.Request.URL.String()
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
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}

	course := models.Course{ID: ID, Title: title}
	if err := course.Update("title", course.Title); err != nil {
		return flashError(
			errDB,
			err,
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}

	c.Flash.Success(c.Message("course.title.change.success",
		course.Title,
		course.ID,
	))
	return c.Redirect(c.Session["currPath"])
}

/*ChangeSubtitle changes the subtitle.
- Roles: creator and editors of the course */
func (c EditCourse) ChangeSubtitle(ID int, subtitle string) revel.Result {

	c.Log.Debug("change subtitle", "ID", ID, "subtitle", subtitle)

	//NOTE: the interceptor assures that the course ID is valid
	course := models.Course{ID: ID, Subtitle: sql.NullString{"", false}}

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
				c.Session["currPath"].(string),
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
			c.Session["currPath"].(string),
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
	return c.Redirect(c.Session["currPath"])
}

/*ChangeTimestamp changes the specified timestamp.
- Roles: creator and editors of the course */
func (c EditCourse) ChangeTimestamp(ID int, date, time, timestampType string) revel.Result {

	c.Log.Debug("change timestamp", "ID", ID, "date", date,
		"time", time, "timestampType", timestampType)

	//NOTE: the interceptor assures that the course ID is valid
	timestamp := date + " " + time
	if timestamp != " " || timestampType != "unsubscribeend" { //only the unsubscribeend can be null
		c.Validation.Required(date).
			MessageKey("validation.invalid.date")
		c.Validation.Required(time).
			MessageKey("validation.invalid.time")

		c.Validation.Check(timestamp,
			models.IsTimestamp{},
		).MessageKey("validation.invalid.timestamp")
	}
	//TODO: if edit, get course, set new timestamp value and validate

	if c.Validation.HasErrors() {
		return flashError(
			errValidation,
			nil,
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}

	course := models.Course{ID: ID}
	var err error

	//update the timestamp
	switch timestampType {

	case "enrollmentstart":
		course.EnrollmentStart = timestamp
		err = course.Update("enrollmentstart", course.EnrollmentStart)

	case "enrollmentend":
		course.EnrollmentEnd = timestamp
		err = course.Update("enrollmentend", course.EnrollmentEnd)

	case "unsubscribeend":
		course.UnsubscribeEnd = sql.NullString{timestamp, false}
		if timestamp != " " {
			course.UnsubscribeEnd.Valid = true
		}
		err = course.Update("unsubscribeend", course.UnsubscribeEnd)

	case "expirationdate":
		course.ExpirationDate = timestamp
		err = course.Update("expirationdate", course.ExpirationDate)

	default:
		return flashError(
			errContent,
			err,
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}

	if err != nil {
		return flashError(
			errDB,
			err,
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}

	if timestamp != " " {
		c.Flash.Success(c.Message("course."+timestampType+".change.success",
			timestamp,
			course.ID,
		))
	} else {
		c.Flash.Success(c.Message("course."+timestampType+".delete.success",
			course.ID,
		))
	}
	return c.Redirect(c.Session["currPath"])
}

/*ChangeUserList adds a user to the user list of a course.
- Roles: creator and editors of the course */
func (c EditCourse) ChangeUserList(ID, userID int, listType string) revel.Result {

	c.Log.Debug("add user to user list", "ID", ID, "userID", userID, "listType", listType)

	//NOTE: the interceptor assures that the course ID is valid
	c.Validation.Required(userID).
		MessageKey("validation.missing.userID")

	if listType != "blacklist" && listType != "whitelist" &&
		listType != "instructor" && listType != "editor" {
		c.Validation.ErrorKey("validation.invalid.params")
	}
	//TODO: if edit, get course, set new timestamp value and validate

	if c.Validation.HasErrors() {
		return flashError(
			errValidation,
			nil,
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}

	entry := models.UserListEntry{UserID: userID, CourseID: ID}
	if err := entry.Insert(listType); err != nil {
		return flashError(
			errDB,
			err,
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}

	//NOTE: if the course is active, the user should get a notification e-mail

	c.Flash.Success(c.Message("course."+listType+".change.success",
		entry.EMail,
		entry.CourseID,
	))
	return c.Redirect(c.Session["currPath"])
}

/*DeleteFromUserList removes a from the user list of a course.
- Roles: creator and editors of the course */
func (c EditCourse) DeleteFromUserList(ID, userID int, listType string) revel.Result {

	c.Log.Debug("delete user from user list", "ID", ID, "userID", userID, "listType", listType)

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
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}

	entry := models.UserListEntry{UserID: userID, CourseID: ID}
	if err := entry.Delete(listType); err != nil {
		return flashError(
			errDB,
			err,
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}

	//NOTE: if the course is active, the user should get a notification e-mail

	c.Flash.Success(c.Message("course."+listType+".delete.success",
		ID,
	))
	return c.Redirect(c.Session["currPath"])
}

/*ChangeViewMatrNr toggles the matriculation number restrictions for an editor/instructor.
- Roles: creator and editors of the course */
func (c EditCourse) ChangeViewMatrNr(ID, userID int, listType string, option bool) revel.Result {

	c.Log.Debug("update user in user list", "ID", ID, "userID", userID,
		"listType", listType, "option", option)

	//NOTE: the interceptor assures that the course ID is valid
	c.Validation.Required(userID).
		MessageKey("validation.missing.userID")

	if listType != "instructor" && listType != "editor" {
		c.Validation.ErrorKey("validation.invalid.params")
	}

	if c.Validation.HasErrors() {
		return flashError(
			errValidation,
			nil,
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}

	entry := models.UserListEntry{UserID: userID, CourseID: ID, ViewMatrNr: option}
	if err := entry.Update(listType); err != nil {
		return flashError(
			errDB,
			err,
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}

	//NOTE: if the course is active, the user should get a notification e-mail

	c.Flash.Success(c.Message("course.matr.nr.change.success",
		entry.EMail,
		entry.CourseID,
	))
	return c.Redirect(c.Session["currPath"])
}

/*ChangeVisibility toggles the visibility of a course.
- Roles: creator and editors of the course */
func (c EditCourse) ChangeVisibility(ID int, option bool) revel.Result {

	c.Log.Debug("update course visibility", "ID", ID, "option", option)

	//NOTE: the interceptor assures that the course ID is valid

	course := models.Course{ID: ID, Visible: option}
	if err := course.Update("visible", course.Visible); err != nil {
		return flashError(
			errDB,
			err,
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}

	c.Flash.Success(c.Message("course.visibility.change.success",
		course.ID,
	))
	return c.Redirect(c.Session["currPath"])
}

/*ChangeText changes the text of the provided column.
- Roles: creator and editors of the course */
func (c EditCourse) ChangeText(ID int, textType, data string) revel.Result {

	c.Log.Debug("change text field", "ID", ID, "textType", textType, "data", data)

	return c.Redirect(c.Session["currPath"])
}

/*ChangeGroup changes the group of a course.
- Roles: creator and editors of the course. */
func (c EditCourse) ChangeGroup(ID, parentID int) revel.Result {

	c.Log.Debug("change group", "ID", ID, "parentID", parentID)

	//NOTE: the interceptor assures that the course ID is valid
	c.Validation.Required(parentID).
		MessageKey("validation.invalid.params")

	if c.Validation.HasErrors() {
		return flashError(
			errValidation,
			nil,
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}

	course := models.Course{
		ID:       ID,
		ParentID: sql.NullInt32{Int32: int32(parentID), Valid: true},
	}
	if err := course.Update("parentid", course.ParentID); err != nil {
		return flashError(
			errDB,
			err,
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}

	c.Flash.Success(c.Message("course.group.change.success",
		course.ID,
	))
	return c.Redirect(c.Session["currPath"])
}
