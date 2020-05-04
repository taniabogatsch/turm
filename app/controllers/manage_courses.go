package controllers

import (
	"strconv"
	"turm/app/models"

	"github.com/revel/revel"
)

/*ActiveCourses renders all active courses of the creator.
- Roles: creator, editors and instructors */
func (c ManageCourses) ActiveCourses() revel.Result {

	c.Log.Debug("render active courses", "url", c.Request.URL)
	c.Session["callPath"] = c.Request.URL.String()
	c.Session["currPath"] = c.Request.URL.String()
	c.ViewArgs["tabName"] = c.Message("creator.tab")

	return c.Render()
}

/*GetActiveCourses renders all active courses of the current user.
- Roles: creator, editors and instructors */
func (c ManageCourses) GetActiveCourses() revel.Result {

	c.Log.Debug("render active courses")

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

	var creator models.CourseList
	if err := creator.GetByUserID(&userID, "creator", true, false); err != nil {
		return renderError(
			err,
			c.Controller,
		)
	}

	var editor models.CourseList
	if err := editor.GetByUserID(&userID, "editor", true, false); err != nil {
		return renderError(
			err,
			c.Controller,
		)
	}

	var instructor models.CourseList
	if err := instructor.GetByUserID(&userID, "instructor", true, false); err != nil {
		return renderError(
			err,
			c.Controller,
		)
	}

	return c.Render(creator, editor, instructor)
}

/*Drafts renders all drafts.
- Roles: creator and editors */
func (c ManageCourses) Drafts() revel.Result {

	c.Log.Debug("render drafts page", "url", c.Request.URL)
	c.Session["callPath"] = c.Request.URL.String()
	c.Session["currPath"] = c.Request.URL.String()
	c.ViewArgs["tabName"] = c.Message("creator.tab")

	return c.Render()
}

/*GetDrafts renders all inactive courses of the current user.
- Roles: creator and editors */
func (c ManageCourses) GetDrafts() revel.Result {

	c.Log.Debug("render drafts")

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

	var creator models.CourseList
	if err := creator.GetByUserID(&userID, "creator", false, false); err != nil {
		return renderError(
			err,
			c.Controller,
		)
	}

	var editor models.CourseList
	if err := editor.GetByUserID(&userID, "editor", false, false); err != nil {
		return renderError(
			err,
			c.Controller,
		)
	}

	return c.Render(creator, editor)
}

/*NewCourseModal renders the modal starting the creation of a new course.
It provides all previous courses of the user (creator or editor) that can
be used as drafts.
- Roles: creator */
func (c ManageCourses) NewCourseModal() revel.Result {

	c.Log.Debug("get new course modal")

	//TODO: get user ID from session
	//TODO: render all possible course drafts

	return c.Render()
}

/*NewCourse creates a new inactive course according to the specified parameters.
- Roles: creator */
func (c ManageCourses) NewCourse(param models.NewCourseParam, msg string) revel.Result {

	c.Log.Debug("create a new course", "param", param)

	param.Validate(c.Validation)
	if c.Validation.HasErrors() {
		return flashError(
			errValidation,
			nil,
			c.Session["currPath"].(string),
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
			c.Session["currPath"].(string),
			c.Controller,
			"",
		)
	}

	var course models.Course
	if param.Option == models.BLANK {

		//create a new blank course
		c.Log.Debug("insert blank course")
		err = course.NewBlank(&creatorID, &param.Title)

	} else if param.Option == models.DRAFT {

		//TODO

	} else {

		//TODO

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

	c.Flash.Success(c.Message("creator.opened.new.course",
		course.Title,
		course.ID,
	))
	return c.Redirect(EditCourse.OpenCourse, course.ID)
}
