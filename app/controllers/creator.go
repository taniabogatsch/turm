package controllers

import (
	"turm/app/models"

	"github.com/revel/revel"
)

/*Activate a course draft.
- Roles: creator of the course */
func (c Creator) Activate(ID int) revel.Result {

	c.Log.Debug("activate course", "ID", ID)

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

	if err := course.Update("active", true); err != nil {
		return flashError(
			errDB,
			err,
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}

	c.Flash.Success(c.Message("creator.course.activated",
		course.Title,
		course.ID,
	))
	//TODO: redirect to course
	return c.Redirect(ManageCourses.ActiveCourses)
}

/*Delete a course (draft).
- Roles: creator of the course */
func (c Creator) Delete(ID int) revel.Result {

	c.Log.Debug("delete course (draft)", "ID", ID)

	//NOTE: the interceptor assures that the course ID is valid

	course := models.Course{ID: ID}
	if valid, err := course.Delete(); err == nil && !valid {
		c.Validation.ErrorKey("validation.invalid.delete.course")
		return flashError(
			errValidation,
			nil,
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	} else if err != nil {
		return flashError(
			errDB,
			err,
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
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
			errDB,
			err,
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}

	c.Flash.Success(c.Message("creator.course.duplicated",
		course.Title,
		course.ID,
	))
	return c.Redirect(c.Session["currPath"])
}
