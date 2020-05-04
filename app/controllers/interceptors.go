package controllers

import (
	"strconv"
	"turm/app"
	"turm/app/models"

	"github.com/revel/revel"
)

//general intercepts each revel controller.
//It sets the service e-mail, the languages, the current language,
//the call path (if not set) and resets the logout timer.
func general(c *revel.Controller) revel.Result {

	c.Log.Debug("executing general interceptor")

	c.ViewArgs["serviceEMail"] = app.ServiceEMail
	c.ViewArgs["languages"] = app.Languages

	//set language
	if c.Session["currentLocale"] == nil {
		//set default language as set in the config file
		c.Log.Debug("setting current locale")
		c.Session.Set("currentLocale", app.DefaultLanguage)
	}
	c.ViewArgs["currentLocale"] = c.Session["currentLocale"]
	c.Request.Locale = c.Session["currentLocale"].(string)

	//ensure that callPath and currPath is not nil
	if c.Session["callPath"] == nil {
		c.Log.Debug("setting call path")
		c.Session["callPath"] = "/"
	}
	if c.Session["currPath"] == nil {
		c.Log.Debug("setting curr path")
		c.Session["currPath"] = "/"
	}

	//reset logout timer
	c.Session.SetDefaultExpiration()
	if c.Session["stayLoggedIn"] != nil {
		if c.Session["stayLoggedIn"] == "true" {
			c.Session.SetNoExpiration()
		}
	}

	//TODO: is a user is logged in, render all courses of that user for the navigation bar

	return nil
}

//authAdmin prevents unauthorized access to controllers of type Admin.
func (c Admin) authAdmin() revel.Result {

	//authorizes all ADMINs with activated accounts
	if c.Session["role"] != nil && c.Session["notActivated"] == nil {
		if c.Session["role"].(string) == models.ADMIN.String() {
			return nil
		}
	}

	c.Flash.Error(c.Message("intercept.invalid.action"))
	return c.Redirect(App.Index)
}

//authApp prevents unauthorized access to controllers of type App.
func (c App) authApp() revel.Result {

	if c.Session["notActivated"] != nil {
		if c.MethodName != "ChangeLanguage" {
			c.Flash.Error(c.Message("intercept.invalid.action"))
			return c.Redirect(User.Logout)
		}
	}

	return nil
}

//authManageCourses prevents unauthorized access to controllers of type ManageCourses.
func (c ManageCourses) authManageCourses() revel.Result {

	if c.Session["role"] != nil && c.Session["notActivated"] == nil &&
		c.Session["userID"] != nil && c.Session["isEditor"] != nil &&
		c.Session["isInstructor"] != nil { //prevent nil references

		//authorize admins and creators
		if c.Session["role"].(string) == models.ADMIN.String() ||
			c.Session["role"].(string) == models.CREATOR.String() {
			return nil
		}

		//editors and instructors are not authorized to create new courses
		if c.MethodName != "NewCourseModal" && c.MethodName != "NewCourse" {
			if c.Session["isEditor"].(string) == "true" {
				return nil
			}

			//instructors are only allowed to see active and inactive courses
			if c.Session["isInstructor"].(string) == "true" && (c.MethodName == "ActiveCourses" ||
				c.MethodName == "GetActiveCourses") {
				return nil
			}
		}
	}

	c.Flash.Error(c.Message("intercept.invalid.action"))
	return c.Redirect(App.Index)
}

//authCreator prevents unauthorized access to controllers of type Creator.
func (c Creator) authCreator() revel.Result {

	if authorized, err := evalEditAuth(c.Controller, "onlyCreator"); err == nil && authorized {
		return nil
	} else if err != nil {
		return flashError(
			errTypeConv,
			err,
			"/",
			c.Controller,
			"",
		)
	}

	c.Flash.Error(c.Message("intercept.invalid.action"))
	return c.Redirect(App.Index)
}

//authEditCourse prevents unauthorized access to controllers of type EditCourse.
func (c EditCourse) authEditCourse() revel.Result {

	if authorized, err := evalEditAuth(c.Controller, "course"); err == nil && authorized {
		return nil
	} else if err != nil {
		return flashError(
			errTypeConv,
			err,
			"/",
			c.Controller,
			"",
		)
	}

	c.Flash.Error(c.Message("intercept.invalid.action"))
	return c.Redirect(App.Index)
}

//authEditEvent prevents unauthorized access to controllers of type EditEvent.
func (c EditEvent) authEditEvent() revel.Result {

	if authorized, err := evalEditAuth(c.Controller, "event"); err == nil && authorized {
		return nil
	} else if err != nil {
		return flashError(
			errTypeConv,
			err,
			"/",
			c.Controller,
			"",
		)
	}

	c.Flash.Error(c.Message("intercept.invalid.action"))
	return c.Redirect(App.Index)
}

//authEditMeeting prevents unauthorized access to controllers of type EditMeeting.
func (c EditMeeting) authEditMeeting() revel.Result {

	if authorized, err := evalEditAuth(c.Controller, "meeting"); err == nil && authorized {
		return nil
	} else if err != nil {
		return flashError(
			errTypeConv,
			err,
			"/",
			c.Controller,
			"",
		)
	}

	c.Flash.Error(c.Message("intercept.invalid.action"))
	return c.Redirect(App.Index)
}

//authUser prevents unauthorized access to controllers of type User.
func (c User) authUser() revel.Result {

	//all
	if c.MethodName == "Logout" || c.MethodName == "NewPassword" ||
		c.MethodName == "ActivationPage" || c.MethodName == "VerifyActivationCode" {
		return nil
	}

	loggedIn := false
	if c.Session["userID"] == nil { //not logged in users
		if c.MethodName == "LoginPage" || c.MethodName == "Login" ||
			c.MethodName == "RegistrationPage" || c.MethodName == "Registration" ||
			c.MethodName == "NewPasswordPage" {
			return nil
		}

	} else { //logged in users
		if c.MethodName == "SetPrefLanguage" || c.MethodName == "PrefLanguagePage" {
			return nil
		}

		//logged in and not activated users
		if c.Session["notActivated"] != nil {
			if c.MethodName == "NewActivationCode" {
				return nil
			}
		}
		loggedIn = true
	}

	if loggedIn {
		return c.Redirect(User.Logout)
	}
	c.Flash.Error(c.Message("intercept.invalid.action"))
	return c.Redirect(App.Index)
}

//evalEditAuth evaluates if a user is authorized to edit a course/event/meeting.
func evalEditAuth(c *revel.Controller, table string) (authorized bool, err error) {

	if c.Session["role"] != nil && c.Session["notActivated"] == nil &&
		c.Session["userID"] != nil { //prevent nil references

		//authorize admins
		if c.Session["role"].(string) == models.ADMIN.String() {
			return true, nil
		}

		IDStr := c.Params.Query.Get("ID") //GET request
		if IDStr == "" {
			IDStr = c.Params.Form.Get("ID") //POST request
		}

		//get the ID
		ID, err := strconv.Atoi(IDStr)
		if err != nil {
			c.Log.Error("failed to parse ID from parameter", "query",
				c.Params.Query.Get("ID"), "form", c.Params.Form.Get("ID"),
				"error", err.Error())
			return false, err
		}

		//only creators and editors of the specified course are allowed to edit it
		var user models.User
		userID := c.Session["userID"].(string)

		authorized, err = user.AuthorizedToEdit(&userID, &table, &ID)
		return authorized, err
	}
	return
}
