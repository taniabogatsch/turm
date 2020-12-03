package controllers

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"turm/app/models"

	"github.com/revel/revel"
)

/*Open an already existing course for modification, etc.
- Roles: creator and editors of this course. */
func (c Edit) Open(ID int) revel.Result {

	c.Log.Debug("open course", "ID", ID)

	//NOTE: the interceptor assures that the course ID is valid

	//get the course data
	course := models.Course{ID: ID}
	if err := course.Get(nil, true, 0); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	if course.Expired && course.Active {
		c.Flash.Error(c.Message("intercept.invalid.action"))
		return c.Redirect(c.Session["callPath"].(string))
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
func (c Edit) Download(ID int, filename string) revel.Result {

	c.Log.Debug("download course", "ID", ID, "filename", filename)

	//NOTE: the interceptor assures that the course ID is valid

	course := models.Course{ID: ID}
	if err := course.Get(nil, true, 0); err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
	}
	//reset some values
	course.CreatorData = models.User{}
	course.Path = models.Groups{}

	//marshal the course data into json format
	json, err := json.Marshal(course)
	if err != nil {
		return flashError(
			errTypeConv, err, "", c.Controller, "")
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
			errContent, err, "", c.Controller, "")
	}
	defer file.Close()

	//write data to the file
	writer := bufio.NewWriter(file)
	_, err = writer.WriteString(jsonString)
	if err != nil {
		return flashError(
			errContent, err, "", c.Controller, "")
	}
	defer writer.Flush()

	//render the file
	return c.RenderFileName(filepath, revel.Attachment)
}

/*Validate all course data.
- Roles: creator and editors of this course. */
func (c Edit) Validate(ID int) revel.Result {

	c.Log.Debug("validate course", "ID", ID)

	//NOTE: the interceptor assures that the course ID is valid

	course := models.Course{ID: ID}
	if err := course.Get(nil, true, 0); err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
	}

	course.Validate(c.Validation)
	if c.Validation.HasErrors() {
		return flashError(
			errValidation, nil, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("creator.course.valid"))
	return c.Redirect(c.Session["currPath"])
}

/*NewEvent creates a new blank event in a course.
- Roles: creator and editors of this course. */
func (c Edit) NewEvent(ID int, value, eventType string, conf models.EditEMailConfig) revel.Result {

	c.Log.Debug("create a new event", "ID", ID, "value", value,
		"eventType", eventType, "conf", conf)

	//NOTE: the interceptor assures that the course ID is valid

	value = strings.TrimSpace(value)
	c.Validation.Check(value,
		revel.MinSize{3},
		revel.MaxSize{255},
	).MessageKey("validation.invalid.text.short")

	if eventType != "normal" && eventType != "calendar" {
		c.Validation.ErrorKey("validation.invalid.params")
	}

	if c.Validation.HasErrors() {
		return flashError(
			errValidation, nil, "/course/events?ID="+strconv.Itoa(ID),
			c.Controller, "")
	}

	//normal event
	if eventType == "normal" {

		conf.ID = ID
		event := models.Event{CourseID: ID, Title: value}
		if err := event.NewBlank(&conf); err != nil {
			return flashError(
				errDB, err, "/course/events?ID="+strconv.Itoa(ID),
				c.Controller, "")
		}

		//if the course is active, send notification e-mail
		if err := sendEMailsEdit(c.Controller, &conf); err != nil {
			return c.RenderJSON(
				response{Status: ERROR, Msg: c.Message(errEMail.String())})
		}

		c.Flash.Success(c.Message("event.new.success", event.Title, event.ID))

	} else { //calendar event

		event := models.CalendarEvent{CourseID: ID, Title: value}
		if err := event.NewBlank(); err != nil {
			return flashError(
				errDB, err, "/course/calendarEvents?ID="+strconv.Itoa(ID),
				c.Controller, "")
		}

		c.Flash.Success(c.Message("event.new.calendar.success", event.Title, event.ID))
	}

	//reload correct content
	if eventType == "normal" {
		return c.Redirect(Course.Events, ID)
	}
	return c.Redirect(Course.CalendarEvents, ID)
}

/*ChangeTimestamp changes the specified timestamp.
- Roles: creator and editors of the course */
func (c Edit) ChangeTimestamp(ID int, fieldID, date, time string,
	conf models.EditEMailConfig) revel.Result {

	c.Log.Debug("change timestamp", "ID", ID, "date", date, "time", time,
		"fieldID", fieldID, "conf", conf)

	//NOTE: the interceptor assures that the course ID is valid

	timestamp := date + " " + time
	valid := (timestamp != " ")
	if valid || fieldID != "unsubscribe_end" { //only the unsubscribeend can be null
		c.Validation.Required(date).
			MessageKey("validation.invalid.date")
		c.Validation.Required(time).
			MessageKey("validation.invalid.time")

		c.Validation.Check(timestamp,
			models.IsTimestamp{},
		).MessageKey("validation.invalid.timestamp")
	}

	if c.Validation.HasErrors() {
		return c.RenderJSON(
			response{Status: INVALID, Msg: getErrorString(c.Validation.Errors)})
	}

	if fieldID != "enrollment_start" && fieldID != "enrollment_end" &&
		fieldID != "unsubscribe_end" && fieldID != "expiration_date" {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message("error.undefined")})
	}

	course := models.Course{ID: ID}
	conf.ID = ID
	var err error

	if err = course.UpdateTimestamp(c.Validation, &conf, fieldID,
		timestamp, valid); err != nil {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message(errDB.String())})
	} else if c.Validation.HasErrors() {
		return c.RenderJSON(
			response{Status: INVALID, Msg: getErrorString(c.Validation.Errors)})
	}

	//if the course is active, send notification e-mail
	if fieldID != "expiration_date" {
		if err = sendEMailsEdit(c.Controller, &conf); err != nil {
			return c.RenderJSON(
				response{Status: ERROR, Msg: c.Message(errEMail.String())})
		}
	}

	msg := c.Message("course."+fieldID+".delete.success", course.ID)
	if valid {
		msg = c.Message("course."+fieldID+".change.success", timestamp, course.ID)
	}

	return c.RenderJSON(
		response{Status: SUCCESS, Msg: msg, FieldID: fieldID, Value: strings.TrimSpace(timestamp)})
}

/*ChangeUserList adds a user to the user list of a course.
- Roles: creator and editors of the course */
func (c Edit) ChangeUserList(ID, userID int, listType string) revel.Result {

	c.Log.Debug("add user to user list", "ID", ID, "userID", userID, "listType", listType)

	//NOTE: the interceptor assures that the course ID is valid

	c.Validation.Required(userID).
		MessageKey("validation.missing.userID")

	if listType != "blacklists" && listType != "whitelists" &&
		listType != "instructors" && listType != "editors" {
		c.Validation.ErrorKey("validation.invalid.params")
	}

	if c.Validation.HasErrors() {
		return flashError(
			errValidation, nil, "/course/editorInstructorList?ID="+strconv.Itoa(ID),
			c.Controller, "")
	}

	entry := models.UserListEntry{UserID: userID, CourseID: ID}
	active, data, err := entry.Insert(listType)
	if err != nil {
		return flashError(
			errDB, err, "/course/editorInstructorList?ID="+strconv.Itoa(ID),
			c.Controller, "")
	}

	//if the course is active, the user gets a notification e-mail
	if active {

		err = sendEMail(c.Controller, &data,
			"email.subject.new.course.role",
			"newCourseRole")

		if err != nil {
			return flashError(errEMail, err, "", c.Controller, data.User.EMail)
		}
	}

	c.Flash.Success(c.Message("course."+listType+".change.success",
		entry.EMail,
		entry.CourseID,
	))

	if listType == "instructors" || listType == "editors" {
		return c.Redirect(Course.EditorInstructorList, ID)
	} else if listType == "whitelists" {
		return c.Redirect(Course.Whitelist, ID)
	}
	return c.Redirect(Course.Blacklist, ID)
}

/*DeleteFromUserList removes a from the user list of a course.
- Roles: creator and editors of the course */
func (c Edit) DeleteFromUserList(ID, userID int, listType string) revel.Result {

	c.Log.Debug("delete user from user list", "ID", ID, "userID", userID, "listType", listType)

	//NOTE: the interceptor assures that the course ID is valid

	c.Validation.Required(userID).
		MessageKey("validation.missing.userID")

	if listType != "blacklists" && listType != "whitelists" &&
		listType != "instructors" && listType != "editors" {
		c.Validation.ErrorKey("validation.invalid.params")
	}

	if c.Validation.HasErrors() {
		return flashError(
			errValidation, nil, "/course/editorInstructorList?ID="+strconv.Itoa(ID),
			c.Controller, "")
	}

	entry := models.UserListEntry{UserID: userID, CourseID: ID}
	active, data, err := entry.Delete(listType)
	if err != nil {
		return flashError(
			errDB, err, "/course/editorInstructorList?ID="+strconv.Itoa(ID),
			c.Controller, "")
	}

	//if the course is active, the user gets a notification e-mail
	if active {

		err = sendEMail(c.Controller, &data,
			"email.subject.course.role.deleted",
			"deleteCourseRole")

		if err != nil {
			return flashError(errEMail, err, "", c.Controller, data.User.EMail)
		}
	}

	c.Flash.Success(c.Message("course."+listType+".delete.success", ID))

	if listType == "instructors" || listType == "editors" {
		return c.Redirect(Course.EditorInstructorList, ID)
	} else if listType == "whitelists" {
		return c.Redirect(Course.Whitelist, ID)
	}
	return c.Redirect(Course.Blacklist, ID)
}

/*ChangeViewMatrNr toggles the matriculation number restrictions for an editor/instructor.
- Roles: creator and editors of the course */
func (c Edit) ChangeViewMatrNr(ID, userID int, listType string, option bool) revel.Result {

	c.Log.Debug("update user in user list", "ID", ID, "userID", userID,
		"listType", listType, "option", option)

	//NOTE: the interceptor assures that the course ID is valid

	c.Validation.Required(userID).
		MessageKey("validation.missing.userID")

	if listType != "instructors" && listType != "editors" {
		c.Validation.ErrorKey("validation.invalid.params")
	}

	if c.Validation.HasErrors() {
		return flashError(
			errValidation, nil, "/course/editorInstructorList?ID="+strconv.Itoa(ID),
			c.Controller, "")
	}

	entry := models.UserListEntry{UserID: userID, CourseID: ID, ViewMatrNr: option}
	active, data, err := entry.Update(listType)
	if err != nil {
		return flashError(
			errDB, err, "/course/editorInstructorList?ID="+strconv.Itoa(ID),
			c.Controller, "")
	}

	//if the course is active, the user gets a notification e-mail
	if active {

		err = sendEMail(c.Controller, &data,
			"email.subject.course.role.authorization",
			"changeViewMatrNr")

		if err != nil {
			return flashError(errEMail, err, "", c.Controller, data.User.EMail)
		}
	}

	c.Flash.Success(c.Message("course.matr.nr.change.success", entry.EMail, entry.CourseID))
	return c.Redirect(Course.EditorInstructorList, ID)
}

/*ChangeBool toggles the provided boolean value of a course.
- Roles: creator and editors of the course */
func (c Edit) ChangeBool(ID int, listType string, option bool) revel.Result {

	c.Log.Debug("update bool", "ID", ID, "listType", listType, "option", option)

	//NOTE: the interceptor assures that the course ID is valid

	if listType != "visible" && listType != "only_ldap" {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message("error.undefined")})
	}

	course := models.Course{ID: ID}
	if err := course.Update(nil, listType, option, nil); err != nil {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message(errDB.String())})
	}

	msg := c.Message("course."+listType+".change.success", course.ID)
	return c.RenderJSON(
		response{Status: SUCCESS, Msg: msg, FieldID: listType, Valid: option})
}

/*ChangeText changes the text of the provided column.
- Roles: creator and editors of the course */
func (c Edit) ChangeText(ID int, fieldID, value string, conf models.EditEMailConfig) revel.Result {

	c.Log.Debug("change text value", "ID", ID, "fieldID", fieldID, "value", value,
		"conf", conf)

	//NOTE: the interceptor assures that the course ID is valid

	value = strings.TrimSpace(value)
	valid := (value != "")

	if valid || fieldID == "title" {

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
			return c.RenderJSON(
				response{Status: INVALID, Msg: getErrorString(c.Validation.Errors)})
		}
	}

	if fieldID != "description" && fieldID != "custom_email" &&
		fieldID != "speaker" && fieldID != "title" &&
		fieldID != "subtitle" && fieldID != "fee" {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message("error.undefined")})
	}

	course := models.Course{ID: ID}
	conf.ID = ID
	var err error

	if fieldID == "fee" && valid {
		value = strings.ReplaceAll(value, ",", ".")
		fee, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return c.RenderJSON(
				response{Status: ERROR, Msg: c.Message("error.undefined")})
		}
		err = course.Update(nil, fieldID, sql.NullFloat64{fee, valid}, &conf)

	} else {
		err = course.Update(nil, fieldID, sql.NullString{value, valid}, &conf)
	}

	if err != nil {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message(errDB.String())})
	}

	//if the course is active, send notification e-mail
	if err = sendEMailsEdit(c.Controller, &conf); err != nil {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message(errEMail.String())})
	}

	msg := c.Message("course."+fieldID+".delete.success", course.ID)
	if valid {
		if fieldID == "title" || fieldID == "subtitle" || fieldID == "fee" {
			msg = c.Message("course."+fieldID+".change.success", value, course.ID)
		} else {
			msg = c.Message("course."+fieldID+".change.success", course.ID)
		}
	}

	return c.RenderJSON(
		response{Status: SUCCESS, Msg: msg, FieldID: fieldID, Value: value})
}

/*ChangeGroup changes the group of a course.
- Roles: creator and editors of the course. */
func (c Edit) ChangeGroup(ID, parentID int, conf models.EditEMailConfig) revel.Result {

	c.Log.Debug("change group", "ID", ID, "parentID", parentID)

	//NOTE: the interceptor assures that the course ID is valid

	c.Validation.Required(parentID).
		MessageKey("validation.invalid.params")

	if c.Validation.HasErrors() {
		return flashError(
			errValidation, nil, "/course/path?ID="+strconv.Itoa(ID),
			c.Controller, "")
	}

	course := models.Course{
		ID:       ID,
		ParentID: sql.NullInt32{Int32: int32(parentID), Valid: true},
	}
	if err := course.Update(nil, "parent_id", course.ParentID, &conf); err != nil {
		return flashError(
			errDB, err, "/course/path?ID="+strconv.Itoa(ID),
			c.Controller, "")
	}

	c.Flash.Success(c.Message("course.group.change.success", course.ID))
	return c.Redirect(Course.Path, ID)
}

/*ChangeEnrollLimit changes the enrollment limit of a course.
- Roles: creator and editors of the course */
func (c Edit) ChangeEnrollLimit(ID int, fieldID string, value int) revel.Result {

	c.Log.Debug("change enrollment limit", "ID", ID, "fieldID", fieldID, "value", value)

	//NOTE: the interceptor assures that the course ID is valid

	c.Validation.Check(value,
		revel.Min{0},
		revel.Max{1000000},
	).MessageKey("validation.invalid.int")

	if c.Validation.HasErrors() {
		return c.RenderJSON(
			response{Status: INVALID, Msg: getErrorString(c.Validation.Errors)})
	}

	valid := (value != 0)

	if fieldID != "enroll_limit_events" {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message("error.undefined")})
	}

	course := models.Course{ID: ID}
	err := course.Update(nil, fieldID, sql.NullInt32{int32(value), valid}, nil)
	if err != nil {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message(errDB.String())})
	}

	msg := c.Message("course."+fieldID+".delete.success", course.ID)
	if valid {
		msg = c.Message("course."+fieldID+".change.success", value, course.ID)
	}
	return c.RenderJSON(
		response{Status: SUCCESS, Msg: msg, FieldID: fieldID, Value: strconv.Itoa(value)})
}

/*ChangeRestriction adds/edits a degree/course of study/semester restriction.
- Roles: creator and editors of the course */
func (c Edit) ChangeRestriction(ID int, restriction models.Restriction) revel.Result {

	c.Log.Debug("change enrollment restriction", "ID", ID, "restriction", restriction)

	//NOTE: the interceptor assures that the course ID is valid

	restriction.CourseID = ID
	restriction.Validate(c.Validation)
	if c.Validation.HasErrors() {
		return flashError(
			errValidation, nil, "/course/restrictions?ID="+strconv.Itoa(ID),
			c.Controller, "")
	}

	if restriction.ID == 0 { //insert
		if err := restriction.Insert(nil, 0); err != nil {
			return flashError(
				errDB, err, "/course/restrictions?ID="+strconv.Itoa(ID),
				c.Controller, "")
		}
	} else { //update
		if err := restriction.Update(); err != nil {
			return flashError(
				errDB, err, "/course/restrictions?ID="+strconv.Itoa(ID),
				c.Controller, "")
		}
	}

	c.Flash.Success(c.Message("course.restriction.change.success",
		restriction.CourseID,
	))
	return c.Redirect(Course.Restrictions, ID)
}

/*DeleteRestriction deletes a restriction.
- Roles: creator and editors of this course. */
func (c Edit) DeleteRestriction(ID, restrictionID int) revel.Result {

	c.Log.Debug("delete enrollment restriction", "ID", ID, "restrictionID", restrictionID)

	//NOTE: the interceptor assures that the course ID is valid

	restriction := models.Restriction{ID: restrictionID}
	if err := restriction.Delete(); err != nil {
		return flashError(
			errDB, err, "/course/restrictions?ID="+strconv.Itoa(ID),
			c.Controller, "")
	}

	c.Flash.Success(c.Message("course.restriction.delete.success", ID))
	return c.Redirect(Course.Restrictions, ID)
}

/*SearchUser searches for users for the different user lists.
- Roles: creator and editors of the course. */
func (c Edit) SearchUser(ID int, value, listType string, searchInactive bool) revel.Result {

	c.Log.Debug("search users", "value", value, "searchInactive", searchInactive, "listType", listType)

	//NOTE: the interceptor assures that the course ID is valid

	value = strings.TrimSpace(value)
	c.Validation.Check(value,
		revel.MinSize{3},
		revel.MaxSize{127},
	).MessageKey("validation.invalid.searchValue")

	if listType != "blacklists" && listType != "whitelists" &&
		listType != "instructors" && listType != "editors" {
		c.Validation.ErrorKey("validation.invalid.params")
	}

	if c.Validation.HasErrors() {
		c.Validation.Keep()
		return c.Render()
	}

	var users models.UserList
	if err := users.Search(&value, &listType, &searchInactive, &ID); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	//TODO: do not load matriculation numbers if the user is not allowed to see them

	return c.Render(users, listType)
}
