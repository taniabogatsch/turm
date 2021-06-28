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
- Roles: creator and editors of this course */
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

	//only set these after the course is loaded - TODO: why?
	c.Session["callPath"] = c.Request.URL.String()
	c.Session["currPath"] = c.Request.URL.String()
	c.Session["lastURL"] = c.Request.URL.String()
	c.ViewArgs["tab"] = c.Message("creator.tab")

	return c.Render(course)
}

/*Download a course as JSON.
- Roles: creator of the course */
func (c Edit) Download(ID int, filename string) revel.Result {

	c.Log.Debug("download course", "ID", ID, "filename", filename)
	c.Session["lastURL"] = c.Request.URL.String()

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
		filename = now + " " + course.Title
		filename = strings.ReplaceAll(filename, "/", " ")
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
- Roles: creator and editors of this course */
func (c Edit) Validate(ID int) revel.Result {

	c.Log.Debug("validate course", "ID", ID)
	c.Session["lastURL"] = c.Request.URL.String()

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
func (c Edit) NewEvent(ID int, value, eventType string,
	conf models.EditEMailConfig) revel.Result {

	c.Log.Debug("create a new event", "ID", ID, "value", value,
		"eventType", eventType, "conf", conf)
	c.Session["lastURL"] = c.Request.URL.String()

	//NOTE: the interceptor assures that the course ID is valid

	models.ValidateLength(&value, "validation.invalid.text.short",
		3, 255, c.Validation)

	if eventType != eventTypeNormal && eventType != eventTypeCalendar {
		c.Validation.ErrorKey("validation.invalid.params")
	}

	if c.Validation.HasErrors() {
		return flashError(
			errValidation, nil, "/course/events?ID="+strconv.Itoa(ID),
			c.Controller, "")
	}

	//normal event
	if eventType == eventTypeNormal {

		conf.ID = ID
		event := models.Event{CourseID: ID, Title: value}
		if err := event.NewBlank(&conf); err != nil {
			return flashError(
				errDB, err, "/course/events?ID="+strconv.Itoa(ID),
				c.Controller, "")
		}

		//if the course is active, send notification e-mail
		conf.Field = "email.edit.new.event"
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
	if eventType == eventTypeNormal {
		return c.Redirect(Course.Events, ID)
	}
	return c.Redirect(Course.CalendarEvents, ID)
}

/*ChangeTimestamp changes the specified timestamp.
- Roles: creator and editors of the course */
func (c Edit) ChangeTimestamp(ID int, fieldID, date, time string,
	conf models.EditEMailConfig) revel.Result {

	c.Log.Debug("change timestamp", "ID", ID, "fieldID", fieldID, "date", date,
		"time", time, "conf", conf)
	c.Session["lastURL"] = c.Request.URL.String()

	//NOTE: the interceptor assures that the course ID is valid

	timestamp := date + " " + time
	valid := (timestamp != " ")
	if valid || fieldID != colUnsubscribeEnd { //only the unsubscribeend can be null
		c.Validation.Required(date).
			MessageKey("validation.invalid.date")
		c.Validation.Required(time).
			MessageKey("validation.invalid.time")
	}

	if c.Validation.HasErrors() {
		return c.RenderJSON(
			response{Status: INVALID, Msg: getErrorString(c.Validation.Errors)})
	}

	if fieldID != colEnrollmentStart && fieldID != colEnrollmentEnd &&
		fieldID != colUnsubscribeEnd && fieldID != colExpirationDate {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message("error.undefined")})
	}

	t, err := getTimestamp(timestamp, c.Controller, valid, fieldID)
	if err != nil {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message("error.undefined")})
	}

	course := models.Course{ID: ID}
	conf.ID = ID

	if err = course.UpdateTimestamp(c.Validation, &conf, fieldID,
		t, valid); err != nil {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message(errDB.String())})
	} else if c.Validation.HasErrors() {
		return c.RenderJSON(
			response{Status: INVALID, Msg: getErrorString(c.Validation.Errors)})
	}

	//if the course is active, send notification e-mail
	if fieldID != colExpirationDate {

		conf.Field = "email.edit." + fieldID
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

	c.Log.Debug("add user to user list", "ID", ID, "userID", userID,
		"listType", listType)
	c.Session["lastURL"] = c.Request.URL.String()

	//NOTE: the interceptor assures that the course ID is valid

	c.Validation.Required(userID).
		MessageKey("validation.missing.userID")

	if listType != tabBlacklists && listType != tabWhitelists &&
		listType != tabInstructors && listType != tabEditors {
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

	if listType == tabInstructors || listType == tabEditors {
		return c.Redirect(Course.EditorInstructorList, ID)
	} else if listType == tabWhitelists {
		return c.Redirect(Course.Whitelist, ID)
	}
	return c.Redirect(Course.Blacklist, ID)
}

/*DeleteFromUserList removes a user from the user list of a course.
- Roles: creator and editors of the course */
func (c Edit) DeleteFromUserList(ID, userID int, listType string) revel.Result {

	c.Log.Debug("delete user from user list", "ID", ID, "userID", userID,
		"listType", listType)
	c.Session["lastURL"] = c.Request.URL.String()

	//NOTE: the interceptor assures that the course ID is valid

	c.Validation.Required(userID).
		MessageKey("validation.missing.userID")

	if listType != tabBlacklists && listType != tabWhitelists &&
		listType != tabInstructors && listType != tabEditors {
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

	if listType == tabInstructors || listType == tabEditors {
		return c.Redirect(Course.EditorInstructorList, ID)
	} else if listType == tabWhitelists {
		return c.Redirect(Course.Whitelist, ID)
	}
	return c.Redirect(Course.Blacklist, ID)
}

/*ChangeViewMatrNr toggles the matriculation number restrictions of an editor/instructor.
- Roles: creator and editors of the course */
func (c Edit) ChangeViewMatrNr(ID, userID int, listType string, option bool) revel.Result {

	c.Log.Debug("update user in user list", "ID", ID, "userID", userID,
		"listType", listType, "option", option)
	c.Session["lastURL"] = c.Request.URL.String()

	//NOTE: the interceptor assures that the course ID is valid

	c.Validation.Required(userID).
		MessageKey("validation.missing.userID")

	if listType != tabInstructors && listType != tabEditors {
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
	c.Session["lastURL"] = c.Request.URL.String()

	//NOTE: the interceptor assures that the course ID is valid

	if listType != colVisible && listType != colOnlyLDAP {
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
	c.Session["lastURL"] = c.Request.URL.String()

	//NOTE: the interceptor assures that the course ID is valid

	value = strings.TrimSpace(value)
	valid := (value != "")

	if valid || fieldID == colTitle {

		if fieldID == colTitle || fieldID == colSubtitle {

			models.ValidateLength(&value, "validation.invalid.text",
				3, 511, c.Validation)

		} else if fieldID == colFee {
			c.Validation.Match(value, models.FeePattern).
				MessageKey("validation.invalid.fee")

		} else {

			models.ValidateLength(&value, "validation.invalid.text.area",
				3, 50000, c.Validation)
		}

		if c.Validation.HasErrors() {
			return c.RenderJSON(
				response{Status: INVALID, Msg: getErrorString(c.Validation.Errors)})
		}
	}

	if fieldID != colDescription && fieldID != colCustomEMail &&
		fieldID != colSpeaker && fieldID != colTitle &&
		fieldID != colSubtitle && fieldID != colFee {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message("error.undefined")})
	}

	course := models.Course{ID: ID}
	conf.ID = ID
	var err error

	if fieldID == colFee && valid {

		value = strings.ReplaceAll(value, ",", ".")
		var fee float64

		fee, err = strconv.ParseFloat(value, 64)
		if err != nil {
			return c.RenderJSON(
				response{Status: ERROR, Msg: c.Message("error.undefined")})
		}

		err = course.Update(nil, fieldID, sql.NullFloat64{
			Float64: fee,
			Valid:   valid,
		}, &conf)

	} else {
		err = course.Update(nil, fieldID, sql.NullString{
			String: value,
			Valid:  valid,
		}, &conf)
	}

	if err != nil {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message(errDB.String())})
	}

	//if the course is active, send notification e-mail
	conf.Field = "email.edit." + fieldID
	if err = sendEMailsEdit(c.Controller, &conf); err != nil {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message(errEMail.String())})
	}

	msg := c.Message("course."+fieldID+".delete.success", course.ID)
	if valid {
		if fieldID == colTitle || fieldID == colSubtitle || fieldID == colFee {
			msg = c.Message("course."+fieldID+".change.success", value, course.ID)
		} else {
			msg = c.Message("course."+fieldID+".change.success", course.ID)
		}
	}

	return c.RenderJSON(
		response{Status: SUCCESS, Msg: msg, FieldID: fieldID, Value: value})
}

/*ChangeGroup of a course.
- Roles: creator and editors of the course */
func (c Edit) ChangeGroup(ID, parentID int, conf models.EditEMailConfig) revel.Result {

	c.Log.Debug("change group", "ID", ID, "parentID", parentID,
		"conf", conf)
	c.Session["lastURL"] = c.Request.URL.String()

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
	c.Session["lastURL"] = c.Request.URL.String()

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
	err := course.Update(nil, fieldID, sql.NullInt32{
		Int32: int32(value),
		Valid: valid,
	}, nil)
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

/*ChangeRestriction adds/edits a degree/course of study/semester restriction of a course.
- Roles: creator and editors of the course */
func (c Edit) ChangeRestriction(ID int, restriction models.Restriction) revel.Result {

	c.Log.Debug("change enrollment restriction", "ID", ID, "restriction", restriction)
	c.Session["lastURL"] = c.Request.URL.String()

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

/*DeleteRestriction of a course.
- Roles: creator and editors of this course */
func (c Edit) DeleteRestriction(ID, restrictionID int) revel.Result {

	c.Log.Debug("delete enrollment restriction", "ID", ID, "restrictionID", restrictionID)
	c.Session["lastURL"] = c.Request.URL.String()

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
- Roles: creator and editors of the course */
func (c Edit) SearchUser(ID int, value, listType string, searchInactive bool) revel.Result {

	c.Log.Debug("search users", "ID", ID, "value", value, "listType", listType,
		"searchInactive", searchInactive)
	c.Session["lastURL"] = c.Request.URL.String()

	//NOTE: the interceptor assures that the course ID is valid

	models.ValidateLength(&value, "validation.invalid.searchValue",
		3, 127, c.Validation)

	if listType != tabBlacklists && listType != tabWhitelists &&
		listType != tabInstructors && listType != tabEditors {
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

	//TODO: do not search by matriculation numbers if the user is not allowed to see them

	return c.Render(users, listType)
}
