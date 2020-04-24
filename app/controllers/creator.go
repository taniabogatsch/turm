package controllers

import (
	"strconv"
	"turm/app/models"

	"github.com/revel/revel"
)

/*ActiveCourses renders all active courses of the creator.
- Roles: creator and editors */
func (c Creator) ActiveCourses() revel.Result {

	c.Log.Debug("render active courses page", "url", c.Request.URL)
	c.Session["callPath"] = c.Request.URL.String()
	c.ViewArgs["tabName"] = c.Message("creator.tab")

	return c.Render()
}

/*Drafts renders all drafts.
- Roles: creator and editors */
func (c Creator) Drafts() revel.Result {

	c.Log.Debug("render drafts page", "url", c.Request.URL)
	c.Session["callPath"] = c.Request.URL.String()
	c.ViewArgs["tabName"] = c.Message("creator.tab")

	return c.Render()
}

/*GetDrafts renders all inactive courses of the current user.
- Roles: creator and editors */
func (c Creator) GetDrafts() revel.Result {

	c.Log.Debug("get drafts")

	//get the user
	userID, err := strconv.Atoi(c.Session["userID"].(string))
	if err != nil {
		c.Log.Error("failed to parse userID from session",
			"session", c.Session, "error", err.Error())
		return renderError(
			err,
			c.Controller,
		)
	}

	var created models.CourseList
	if err := created.GetByUserID(&userID, false, false, false); err != nil {
		return renderError(
			err,
			c.Controller,
		)
	}

	//TODO: get courses of which the user is editor

	return c.Render(created)
}

/*NewCourseModal renders the modal starting the creation of a new course.
It provides all previous courses of the user (creator or editor) that can
be used as drafts.
- Roles: creator */
func (c Creator) NewCourseModal() revel.Result {

	c.Log.Debug("get new course modal")

	//TODO: get user ID from session
	//TODO: render all possible course drafts

	return c.Render()
}

/*NewCourse creates a new inactive course according to the specified parameters.
- Roles: creators */
func (c Creator) NewCourse(param models.NewCourseParam, msg string) revel.Result {

	c.Log.Debug("render new course page", "param", param)

	param.Validate(c.Validation)
	if c.Validation.HasErrors() {
		return flashError(
			errValidation,
			nil,
			c.Session["callPath"].(string),
			c.Controller,
			"",
		)
	}

	//get the course creator
	creatorID, err := strconv.Atoi(c.Session["userID"].(string))
	if err != nil {
		c.Log.Error("failed to parse userID from session",
			"session", c.Session, "error", err.Error())
		return flashError(
			errTypeConv,
			err,
			c.Session["callPath"].(string),
			c.Controller,
			"",
		)
	}

	var course models.Course
	if param.Option == models.BLANK {
		c.Log.Debug("insert blank course")
		err = course.NewBlank(&creatorID, &param.Title)
		msg = c.Message("creator.new.blank.success", course.ID)
	} else if param.Option == models.DRAFT {
		//TODO
	} else {
		//TODO
	}
	if err != nil {
		return flashError(
			errDB,
			err,
			c.Session["callPath"].(string),
			c.Controller,
			"",
		)
	}

	return c.Redirect(Creator.OpenCourse, course.ID, msg)
}

/*OpenCourse opens an already existing course for modification, etc.
- Roles: creator and editors of this course. */
func (c Creator) OpenCourse(ID int, msg string) revel.Result {

	c.Log.Debug("course management: open course", "ID", ID, "msg", msg)

	//TODO: param validation

	//get the course data
	course := models.Course{ID: ID}
	if err := course.Get(); err != nil {
		return flashError(
			errDB,
			err,
			c.Session["callPath"].(string),
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
