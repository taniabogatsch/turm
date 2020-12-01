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

	c.ViewArgs["serviceEMail"] = app.Mailer.EMail
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
		if c.Session["stayLoggedIn"] == cTrue {
			c.Session.SetNoExpiration()
		}
	}

	//TODO: is a user is logged in, render all courses of that user for the navigation bar
	//TODO: but do so via an ajax call!

	return nil
}

//auth prevents unauthorized access to controllers of type Admin.
func (c Admin) auth() revel.Result {

	c.Log.Debug("executing auth admin interceptor")

	//authorizes all ADMINs with activated accounts
	if c.Session["role"] != nil && c.Session["notActivated"] == nil {
		if c.Session["role"].(string) == models.ADMIN.String() {
			return nil
		}
	}

	c.Flash.Error(c.Message("intercept.invalid.action"))
	return c.Redirect(App.Index)
}

//auth prevents unauthorized access to controllers of type App.
func (c App) auth() revel.Result {

	c.Log.Debug("executing auth app interceptor")

	if c.Session["notActivated"] != nil {
		if c.MethodName != "ChangeLanguage" {
			c.Flash.Error(c.Message("intercept.invalid.action"))
			return c.Redirect(User.Logout)
		}
	}

	return nil
}

//auth prevents unauthorized access to controllers of type Manage.
func (c Manage) auth() revel.Result {

	c.Log.Debug("executing auth manage courses interceptor")

	if c.Session["role"] != nil && c.Session["notActivated"] == nil &&
		c.Session["userID"] != nil && c.Session["isEditor"] != nil &&
		c.Session["isInstructor"] != nil { //prevent nil references

		//authorize admins and creators
		if c.Session["role"].(string) == models.ADMIN.String() ||
			c.Session["role"].(string) == models.CREATOR.String() {
			return nil
		}

		if c.Session["isEditor"].(string) == cTrue {
			return nil
		}

		//instructors are only allowed to see active and expired courses
		if c.Session["isInstructor"].(string) == cTrue && (c.MethodName == "Active" ||
			c.MethodName == "GetActive" || c.MethodName == "Expired" || c.MethodName == "GetExpired") {
			return nil
		}
	}

	c.Flash.Error(c.Message("intercept.invalid.action"))
	return c.Redirect(App.Index)
}

//auth prevents unauthorized access to controllers of type Course.
func (c Course) auth() revel.Result {

	c.Log.Debug("executing auth course interceptor")

	//only allow a course search if the user is not logged in or
	//if the user account is activated
	if c.MethodName == "Search" {
		if c.Session["notActivated"] == nil {
			return nil
		}
	}

	if authorized, expired, err := evalHasElevatedRights(c.Controller); err != nil {
		return flashError(
			errTypeConv, err, "/", c.Controller, "")
	} else if expired && !authorized {
		c.Flash.Error(c.Message("intercept.course.expired"))
		return c.Redirect(App.Index)
	}

	//only render content if the course is publicly visible or an user is logged in
	if c.Session["userID"] == nil && c.MethodName != "Open" {

		IDStr := c.Params.Query.Get("ID") //GET request
		if IDStr == "" {
			IDStr = c.Params.Form.Get("ID") //POST request
		}

		//get course ID
		ID, err := strconv.Atoi(IDStr)
		if err != nil {
			c.Log.Error("failed to parse ID from parameter", "query",
				c.Params.Query.Get("ID"), "form", c.Params.Form.Get("ID"),
				"error", err.Error())
			return flashError(
				errTypeConv, err, "/", c.Controller, "")
		}

		course := models.Course{ID: ID}
		elem := "course"
		if c.MethodName == "Meetings" {
			//get course ID from meeting
			elem = "event"
		}

		if err = course.GetVisible(elem); err != nil {
			return flashError(
				errDB, err, "/", c.Controller, "")
		}

		if !course.Visible {
			c.Flash.Error(c.Message("intercept.invalid.action"))
			return c.Redirect(App.Index)
		}
	}

	if c.Session["userID"] != nil && (c.MethodName == "Whitelist" ||
		c.MethodName == "Blacklist") {

		if authorized, _, err := evalEditAuth(c.Controller, "courses"); err != nil {
			return flashError(
				errTypeConv, err, "/", c.Controller, "")
		} else if !authorized {
			c.Flash.Error(c.Message("intercept.invalid.action"))
			return c.Redirect(App.Index)
		}
	}

	return nil
}

//auth prevents unauthorized access to controllers of type Creator.
func (c Creator) auth() revel.Result {

	c.Log.Debug("executing auth creator interceptor")

	//admins and creators are authorized to create new courses
	if c.MethodName == "New" && c.Session["role"] != nil {
		if c.Session["role"] == models.ADMIN.String() ||
			c.Session["role"] == models.CREATOR.String() {
			return nil
		}
	}

	authorized, expired, err := evalEditAuth(c.Controller, "onlyCreator")
	if err != nil {
		return flashError(
			errTypeConv, err, "/", c.Controller, "")
	}

	if c.Session["role"] != nil {
		if c.Session["role"] == models.ADMIN.String() {
			authorized = true
		}
	}

	if !authorized {
		c.Flash.Error(c.Message("intercept.invalid.action"))
		return c.Redirect(App.Index)
	}

	if expired && (c.MethodName == "Activate") {
		c.Flash.Error(c.Message("intercept.invalid.action"))
		return c.Redirect(App.Index)
	}

	return nil
}

//auth prevents unauthorized access to controllers of type Edit.
func (c Edit) auth() revel.Result {

	c.Log.Debug("executing auth edit courses interceptor")

	if authorized, expired, err := evalEditAuth(c.Controller, "courses"); err != nil {
		return flashError(
			errTypeConv, err, "/", c.Controller, "")
	} else if expired || !authorized {
		c.Flash.Error(c.Message("intercept.invalid.action"))
		return c.Redirect(App.Index)
	}

	return nil
}

//auth prevents unauthorized access to controllers of type EditEvent.
func (c EditEvent) auth() revel.Result {

	c.Log.Debug("executing auth edit events interceptor")

	if authorized, expired, err := evalEditAuth(c.Controller, "events"); err != nil {
		return flashError(
			errTypeConv, err, "/", c.Controller, "")
	} else if expired || !authorized {
		c.Flash.Error(c.Message("intercept.invalid.action"))
		return c.Redirect(App.Index)
	}

	return nil
}

//auth prevents unauthorized access to controllers of type EditCalendarEvent.
func (c EditCalendarEvent) auth() revel.Result {

	c.Log.Debug("executing auth edit calendar events interceptor")

	if authorized, expired, err := evalEditAuth(c.Controller, "calendar_events"); err != nil {
		return flashError(
			errTypeConv, err, "/", c.Controller, "")
	} else if expired || !authorized {
		c.Flash.Error(c.Message("intercept.invalid.action"))
		return c.Redirect(App.Index)
	}

	return nil
}

//auth prevents unauthorized access to controllers of type EditMeeting.
func (c EditMeeting) auth() revel.Result {

	c.Log.Debug("executing auth edit meetings interceptor")

	if authorized, expired, err := evalEditAuth(c.Controller, "meetings"); err != nil {
		return flashError(
			errTypeConv, err, "/", c.Controller, "")
	} else if expired || !authorized {
		c.Flash.Error(c.Message("intercept.invalid.action"))
		return c.Redirect(App.Index)
	}

	return nil
}

//auth prevents unauthorized access to controllers of type User.
func (c User) auth() revel.Result {

	c.Log.Debug("executing auth user interceptor")

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

		//activated users
		if (c.MethodName == "Profile" || c.MethodName == "ChangePassword") &&
			c.Session["notActivated"] == nil {
			return nil
		}

		//not activated users
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

//auth prevents unauthorized access to controllers of type Enrollment
func (c Enrollment) auth() revel.Result {

	c.Log.Debug("executing auth enrollment interceptor")

	//authorizes all logged in users
	if c.Session["userID"] != nil {
		return nil
	}

	//TODO: only activated accounts

	c.Flash.Error(c.Message("intercept.invalid.action"))
	return c.Redirect(App.Index)
}

//auth prevents unauthorized access to controllers of type Participants.
func (c Participants) auth() revel.Result {

	c.Log.Debug("executing auth participants interceptor")

	if authorized, _, err := evalHasElevatedRights(c.Controller); err != nil {
		return flashError(
			errTypeConv, err, "/", c.Controller, "")
	} else if !authorized {
		c.Flash.Error(c.Message("intercept.invalid.action"))
		return c.Redirect(App.Index)
	}

	if c.MethodName == "SearchUser" || c.MethodName == "Enroll" ||
		c.MethodName == "Unsubscribe" || c.MethodName == "Waitlist" ||
		c.MethodName == "ChangeStatus" {

		fittingEventID, err := evalEventIDFitsCourseID(c.Controller)
		if err != nil {
			return flashError(
				errTypeConv, err, "/", c.Controller, "")
		} else if !fittingEventID {
			c.Flash.Error(c.Message("intercept.invalid.action"))
			return c.Redirect(App.Index)
		}
	}

	return nil
}

//evalEditAuth evaluates if a user is authorized to edit a course/event/meeting.
func evalEditAuth(c *revel.Controller, table string) (authorized, expired bool, err error) {

	user := models.User{}
	if c.Session["userID"] != nil && c.Session["role"] != nil &&
		c.Session["notActivated"] == nil {

		//authorize admins
		if c.Session["role"].(string) == models.ADMIN.String() {
			return true, false, nil
		}

		user.ID, err = getIntFromSession(c, "userID")
		if err != nil {
			return
		}
	}

	//get the course ID
	IDStr := c.Params.Query.Get("ID") //GET request
	if IDStr == "" {
		IDStr = c.Params.Form.Get("ID") //POST request
	}

	//get the course ID
	ID, err := strconv.Atoi(IDStr)
	if err != nil {
		c.Log.Error("failed to parse ID from parameter", "IDStr", IDStr, "error", err.Error())
		return false, false, err
	}

	authorized, expired, err = user.AuthorizedToEdit(&table, &ID)
	return
}

//evalHasElevatedRights evaluates if a user is an instructor, editor, creator or
//admin (of a course).
func evalHasElevatedRights(c *revel.Controller) (authorized, expired bool, err error) {

	//only instructors, creators and editors of the specified course are allowed
	//to manage participants
	user := models.User{}
	if c.Session["userID"] != nil && c.Session["role"] != nil &&
		c.Session["notActivated"] == nil {

		//authorize admins
		if c.Session["role"].(string) == models.ADMIN.String() {
			return true, false, nil
		}

		user.ID, err = getIntFromSession(c, "userID")
		if err != nil {
			return
		}
	}

	//get the course ID
	IDStr := c.Params.Query.Get("ID") //GET request
	if IDStr == "" {
		IDStr = c.Params.Form.Get("ID") //POST request
	}

	//get the course ID
	ID, err := strconv.Atoi(IDStr)
	if err != nil {
		c.Log.Error("failed to parse ID from parameter", "IDStr", IDStr, "error", err.Error())
		return false, false, err
	}

	return user.HasElevatedRights(&ID)
}

//evalEventIDFitsCourseID evaluates whether the provided event is part of the course or not
func evalEventIDFitsCourseID(c *revel.Controller) (fittingEventID bool, err error) {

	//get the course ID
	IDStr := c.Params.Query.Get("ID") //GET request
	if IDStr == "" {
		IDStr = c.Params.Form.Get("ID") //POST request
	}

	//get the course ID
	ID, err := strconv.Atoi(IDStr)
	if err != nil {
		c.Log.Error("failed to parse ID from parameter", "IDStr", IDStr,
			"error", err.Error())
		return false, err
	}

	//get the event ID
	eventIDStr := c.Params.Query.Get("eventID") //GET request
	if eventIDStr == "" {
		eventIDStr = c.Params.Form.Get("eventID") //POST request
	}

	//get the event ID
	eventID, err := strconv.Atoi(eventIDStr)
	if err != nil {
		c.Log.Error("failed to parse event ID from parameter", "eventIDStr",
			eventIDStr, "error", err.Error())
		return false, err
	}

	event := models.Event{ID: eventID}
	return event.BelongsToCourse(&ID)
}
