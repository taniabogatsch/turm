package controllers

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"turm/app/models"

	"github.com/revel/revel"
)

//TODO: notify enrolled users if course is updated

/*OpenCourse opens an already existing course for modification, etc.
- Roles: creator and editors of this course. */
func (c EditCourse) OpenCourse(ID int) revel.Result {

	c.Log.Debug("course management: open course", "ID", ID)

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
	return c.Render(course)
}

/*Download a course as JSON.
- Roles: creator of the course */
func (c EditCourse) Download(ID int, filename string) revel.Result {

	c.Log.Debug("download course", "ID", ID, "filename", filename)

	//NOTE: the interceptor assures that the course ID is valid

	course := models.Course{ID: ID}
	if err := course.Get(); err != nil {
		return flashError(
			errDB,
			err,
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}
	//reset some values
	course.CreatorData = models.User{}
	course.Path = models.Groups{}

	//marshal the course data into json format
	json, err := json.Marshal(course)
	if err != nil {
		return flashError(
			errTypeConv,
			err,
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}
	jsonString := string(json[:])

	//if the user did not provide a custom file name
	if filename == "" {
		now := time.Now().Format(revel.TimeFormats[1])
		filename = strings.ReplaceAll(now+"_"+course.Title, " ", "_")
	}

	//filepath
	filepath := filepath.Join("/tmp", filename+".json")

	//create the file
	file, err := os.Create(filepath)
	if err != nil {
		return flashError(
			errContent,
			err,
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}
	defer file.Close()

	//write data to the file
	writer := bufio.NewWriter(file)
	_, err = writer.WriteString(jsonString)
	if err != nil {
		return flashError(
			errContent,
			err,
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}
	defer writer.Flush()

	//render the file
	return c.RenderFileName(filepath, revel.Attachment)
}

/*Validate all course data.
- Roles: creator and editors of this course. */
func (c EditCourse) Validate(ID int) revel.Result {

	c.Log.Debug("validate course", "ID", ID)

	//NOTE: the interceptor assures that the course ID is valid

	course := models.Course{ID: ID}
	if err := course.Get(); err != nil {
		return flashError(
			errDB,
			err,
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}

	course.Validate(c.Validation)
	if c.Validation.HasErrors() {
		return flashError(
			errValidation,
			nil,
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}

	c.Flash.Success(c.Message("creator.course.valid"))
	return c.Redirect(c.Session["currPath"])
}

/*NewEvent creates a new blank event in a course.
- Roles: creator and editors of this course. */
func (c EditCourse) NewEvent(ID int, fieldID, value string) revel.Result {

	c.Log.Debug("create a new event", "ID", ID, "fieldID", fieldID, "value", value)

	//NOTE: the interceptor assures that the course ID is valid

	value = strings.TrimSpace(value)
	c.Validation.Check(value,
		revel.MinSize{3},
		revel.MaxSize{255},
	).MessageKey("validation.invalid.text.short")

	if c.Validation.HasErrors() {
		return flashError(
			errValidation,
			nil,
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}

	event := models.Event{CourseID: ID, Title: value}
	err := event.NewBlank()
	if err != nil {
		return flashError(
			errDB,
			err,
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}

	c.Flash.Success(c.Message("event.new.success",
		event.Title,
		event.ID,
	))
	return c.Redirect(c.Session["currPath"])
}

/*ChangeTimestamp changes the specified timestamp.
- Roles: creator and editors of the course */
func (c EditCourse) ChangeTimestamp(ID int, fieldID, date, time string) revel.Result {

	c.Log.Debug("change timestamp", "ID", ID, "date", date,
		"time", time, "fieldID", fieldID)

	//NOTE: the interceptor assures that the course ID is valid
	timestamp := date + " " + time
	valid := (timestamp != " ")
	if valid || fieldID != "unsubscribeend" { //only the unsubscribeend can be null
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

	if fieldID != "enrollmentstart" && fieldID != "enrollmentend" &&
		fieldID != "unsubscribeend" && fieldID != "expirationdate" {
		return flashError(
			errContent,
			errors.New("invalid column value"),
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}

	course := models.Course{ID: ID}
	var err error

	if fieldID == "unsubscribeend" {
		err = course.Update(fieldID, sql.NullString{timestamp, valid})
	} else {
		err = course.Update(fieldID, timestamp)
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

	if valid {
		c.Flash.Success(c.Message("course."+fieldID+".change.success",
			timestamp,
			course.ID,
		))
	} else {
		c.Flash.Success(c.Message("course."+fieldID+".delete.success",
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

	//TODO: if the course is active, the user should get a notification e-mail

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

	//TODO: if the course is active, the user should get a notification e-mail

	c.Flash.Success(c.Message("course.matr.nr.change.success",
		entry.EMail,
		entry.CourseID,
	))
	return c.Redirect(c.Session["currPath"])
}

/*ChangeBool toggles the provided boolean value of a course.
- Roles: creator and editors of the course */
func (c EditCourse) ChangeBool(ID int, listType string, option bool) revel.Result {

	c.Log.Debug("update bool", "ID", ID, "listType", listType, "option", option)

	//NOTE: the interceptor assures that the course ID is valid

	if listType != "visible" && listType != "onlyldap" {
		return flashError(
			errContent,
			errors.New("invalid column value"),
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}

	course := models.Course{ID: ID}
	if err := course.Update(listType, option); err != nil {
		return flashError(
			errDB,
			err,
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}

	c.Flash.Success(c.Message("course."+listType+".change.success",
		course.ID,
	))
	return c.Redirect(c.Session["currPath"])
}

/*ChangeText changes the text of the provided column.
- Roles: creator and editors of the course */
func (c EditCourse) ChangeText(ID int, fieldID, value string) revel.Result {

	c.Log.Debug("change text value", "ID", ID, "fieldID", fieldID, "value", value)

	value = strings.TrimSpace(value)
	valid := (value != "")

	//NOTE: the interceptor assures that the course ID is valid
	if value != "" || fieldID == "title" {

		if fieldID == "title" || fieldID == "subtitle" {
			c.Validation.Check(value,
				revel.MinSize{3},
				revel.MaxSize{511},
			).MessageKey("validation.invalid.text")

		} else if fieldID == "fee" {
			c.Validation.Match(value, models.FeePattern).
				MessageKey("validation.invalid.fee")

		} else {
			c.Validation.Check(value,
				revel.MinSize{3},
				revel.MaxSize{50000},
			).MessageKey("validation.invalid.text.area")
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
	}

	if fieldID != "description" && fieldID != "customemail" &&
		fieldID != "speaker" && fieldID != "title" &&
		fieldID != "subtitle" && fieldID != "fee" {
		return flashError(
			errContent,
			errors.New("invalid column value"),
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}

	course := models.Course{ID: ID}
	var err error

	if fieldID == "fee" && valid {
		value = strings.ReplaceAll(value, ",", ".")
		fee, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return flashError(
				errContent,
				err,
				c.Session["currPath"].(string),
				c.Controller,
				"",
			)
		}
		err = course.Update(fieldID, sql.NullFloat64{fee, valid})

	} else {
		err = course.Update(fieldID, sql.NullString{value, valid})
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

	if valid {
		if fieldID == "title" || fieldID == "subtitle" || fieldID == "fee" {
			c.Flash.Success(c.Message("course."+fieldID+".change.success",
				value,
				course.ID,
			))
		} else {
			c.Flash.Success(c.Message("course."+fieldID+".change.success",
				course.ID,
			))
		}
	} else {
		c.Flash.Success(c.Message("course."+fieldID+".delete.success",
			course.ID,
		))
	}
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

/*ChangeEnrollLimit changes the enrollment limit of a course.
- Roles: creator and editors of the course */
func (c EditCourse) ChangeEnrollLimit(ID int, fieldID string, value int) revel.Result {

	c.Log.Debug("change enrollment limit", "ID", ID, "fieldID", fieldID, "value", value)

	//NOTE: the interceptor assures that the course ID is valid
	//NOTE: no validation is required; set to null, if value is 0

	valid := (value != 0)

	if fieldID != "enrolllimitevents" {
		return flashError(
			errContent,
			errors.New("invalid column value"),
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}

	course := models.Course{ID: ID}
	err := course.Update(fieldID, sql.NullInt32{int32(value), valid})
	if err != nil {
		return flashError(
			errDB,
			err,
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}

	if valid {
		c.Flash.Success(c.Message("course."+fieldID+".change.success",
			value,
			course.ID,
		))
	} else {
		c.Flash.Success(c.Message("course."+fieldID+".delete.success",
			course.ID,
		))
	}
	return c.Redirect(c.Session["currPath"])
}

/*SearchUser searches for users for the different user lists.
- Roles: creator and editors of the course. */
func (c EditCourse) SearchUser(ID int, value, listType string, searchInactive bool) revel.Result {

	c.Log.Debug("search users", "value", value, "searchInactive", searchInactive, "listType", listType)

	value = strings.TrimSpace(value)
	c.Validation.Check(value,
		revel.MinSize{3},
		revel.MaxSize{127},
	).MessageKey("validation.invalid.searchValue")

	if listType != "blacklist" && listType != "whitelist" &&
		listType != "instructor" && listType != "editor" {
		c.Validation.ErrorKey("validation.invalid.params")
	}

	//NOTE: the interceptor assures that the course ID is valid

	if c.Validation.HasErrors() {
		c.Validation.Keep()
		return c.Render()
	}

	var users models.UserList
	if err := users.Search(&value, &listType, &searchInactive, &ID); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	return c.Render(users, listType)
}