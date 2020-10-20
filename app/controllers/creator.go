package controllers

import (
	"time"
	"turm/app/models"

	"github.com/revel/revel"
)

/*Activate a course draft.
- Roles: creator of the course */
func (c Creator) Activate(ID int) revel.Result {

	c.Log.Debug("activate course", "ID", ID)

	//NOTE: the interceptor assures that the course ID is valid

	course := models.Course{ID: ID}
	if invalid, err := course.Activate(c.Validation); err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
	} else if invalid {
		return flashError(
			errValidation, nil, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("creator.course.activated",
		course.Title,
		course.ID,
	))

	return c.Redirect(Manage.Active)
}

/*Delete a course (draft).
- Roles: creator of the course */
func (c Creator) Delete(ID int) revel.Result {

	c.Log.Debug("delete course (draft)", "ID", ID)

	//NOTE: the interceptor assures that the course ID is valid

	course := models.Course{ID: ID}
	if valid, err := course.Delete(); err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
	} else if !valid {
		c.Validation.ErrorKey("validation.invalid.delete.course")
		return flashError(
			errValidation, nil, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("creator.course.deleted",
		ID,
	))
	return c.Redirect(c.Session["currPath"])
}

/*Duplicate a course draft.
- Roles: creator of the course */
func (c Creator) Duplicate(ID int, title string) revel.Result {

	c.Log.Debug("duplicate course draft", "ID", ID)

	//NOTE: the interceptor assures that the course ID is valid

	course := models.Course{ID: ID, Title: title}
	if err := course.Duplicate(); err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("creator.course.duplicated",
		course.Title,
		course.ID,
	))
	return c.Redirect(c.Session["currPath"])
}

/*Expire an active course.
- Roles: creator of the course */
func (c Creator) Expire(ID int) revel.Result {

	c.Log.Debug("expire course", "ID", ID)

	//NOTE: the interceptor assures that the course ID is valid

	now := time.Now().Add(-time.Minute * 1).Format(revel.TimeFormats[0])

	course := models.Course{ID: ID}
	if err := course.Update(nil, "expiration_date", now); err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("creator.course.expired",
		course.ID,
	))
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
		return flashError(
			errValidation, nil, "", c.Controller, "")
	}

	//get the course creator
	creatorID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		return flashError(
			errTypeConv, err, "", c.Controller, "")
	}

	if param.Option == models.BLANK {

		c.Log.Debug("insert blank course")
		err = course.NewBlank(&creatorID, &param.Title)

	} else if param.Option == models.DRAFT {

		//TODO
		return c.Redirect(Manage.Drafts)

	} else {

		c.Log.Debug("insert uploaded course")
		err = course.Insert(&creatorID, &param.Title)

	}
	if err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("creator.opened.new.course",
		course.Title,
		course.ID,
	))
	return c.Redirect(Edit.Open, course.ID)
}
