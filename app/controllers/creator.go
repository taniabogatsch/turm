package controllers

import (
	"database/sql"
	"strings"
	"time"
	"turm/app/models"

	"github.com/revel/revel"
)

/*Activate a course draft.
- Roles: creator of the course
- Roles: creator */
func (c Creator) Activate(ID int) revel.Result {

	c.Log.Debug("activate course", "ID", ID)

	//NOTE: the interceptor assures that the course ID is valid

	course := models.Course{ID: ID}
	invalid, users, err := course.Activate(c.Validation)

	if err != nil {
		return flashError(errDB, err, "", c.Controller, "")
	} else if invalid {
		return flashError(errValidation, nil, "", c.Controller, "")
	}

	//send notification e-mail to editors/instructors
	for _, data := range users {

		err = sendEMail(c.Controller, &data,
			"email.subject.new.course.role",
			"newCourseRole")

		if err != nil {
			return flashError(errEMail, err, "", c.Controller, data.User.EMail)
		}
	}

	c.Flash.Success(c.Message("creator.course.activated",
		course.Title,
		course.ID,
	))

	return c.Redirect(Manage.Active)
}

/*Delete a course (draft).
- Roles: creator of the course
- Roles: creator */
func (c Creator) Delete(ID int) revel.Result {

	c.Log.Debug("delete course (draft)", "ID", ID)

	//NOTE: the interceptor assures that the course ID is valid

	course := models.Course{ID: ID}
	if valid, err := course.Delete(); err != nil {
		return flashError(errDB, err, "", c.Controller, "")
	} else if !valid {
		c.Validation.ErrorKey("validation.invalid.delete.course")
		return flashError(errValidation, nil, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("creator.course.deleted", ID))
	return c.Redirect(c.Session["currPath"])
}

/*Duplicate a course draft.
- Roles: creator of the course
- Roles: creator */
func (c Creator) Duplicate(ID int, title string) revel.Result {

	c.Log.Debug("duplicate course draft", "ID", ID)

	//NOTE: the interceptor assures that the course ID is valid

	//get the course creator
	creatorID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		return flashError(errTypeConv, err, "", c.Controller, "")
	}

	course := models.Course{ID: ID, Title: title,
		Creator: sql.NullInt32{int32(creatorID), true}}
	if err := course.Duplicate(); err != nil {
		return flashError(errDB, err, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("creator.course.duplicated",
		course.Title,
		course.ID))
	return c.Redirect(Manage.Drafts)
}

/*Expire an active course.
- Roles: creator of the course
- Roles: creator */
func (c Creator) Expire(ID int) revel.Result {

	c.Log.Debug("expire course", "ID", ID)

	//NOTE: the interceptor assures that the course ID is valid

	now := time.Now().Add(-time.Minute * 1).Format(revel.TimeFormats[0])

	course := models.Course{ID: ID}
	if err := course.Update(nil, "expiration_date", now, nil); err != nil {
		return flashError(errDB, err, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("creator.course.expired", course.ID))
	return c.Redirect(Manage.Expired)
}

/*New creates a new inactive course according to the specified parameters.
- Roles: creator */
func (c Creator) New(param models.NewCourseParam, file []byte) revel.Result {

	c.Log.Debug("create a new course", "param", param, "file", string(file))
	param.JSON = file

	var course models.Course
	param.Validate(c.Validation, &course)
	if c.Validation.HasErrors() {
		return flashError(errValidation, nil, "", c.Controller, "")
	}

	//get the course creator
	creatorID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		return flashError(errTypeConv, err, "", c.Controller, "")
	}

	course.ID = param.CourseID
	course.Creator = sql.NullInt32{int32(creatorID), true}
	course.Title = param.Title

	if param.Option == models.BLANK {

		c.Log.Debug("insert blank course")
		err = course.NewBlank()

	} else if param.Option == models.DRAFT {

		c.Log.Debug("insert draft")
		err = course.InsertFromDraft(c.Validation)
		if c.Validation.HasErrors() {
			return flashError(errValidation, nil, "", c.Controller, "")
		}

	} else {

		c.Log.Debug("insert uploaded course")
		err = course.Insert()

	}

	if err != nil {
		return flashError(errDB, err, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("creator.opened.new.course",
		course.Title,
		course.ID))
	return c.Redirect(Edit.Open, course.ID)
}

/*Search returns all courses matching a search value for which the user has edit rights. */
func (c Creator) Search(value string) revel.Result {

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

	//get the userID
	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		return flashError(errTypeConv, err, "", c.Controller, "")
	}

	var courses models.CourseList
	err = courses.SearchForDrafts(value, userID, c.Session["role"].(string))
	if err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	return c.Render(courses)
}
