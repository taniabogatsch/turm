package controllers

import (
	"errors"
	"strings"
	"turm/app"
	"turm/app/models"

	"github.com/revel/revel"
)

/*UserManagement renders the user management page.
- Roles: admin */
func (c Admin) UserManagement() revel.Result {

	c.Log.Debug("render user management page", "url", c.Request.URL)
	c.Session["callPath"] = c.Request.URL.String()
	c.ViewArgs["tabName"] = c.Message("admin.tab")

	return c.Render()
}

/*RoleManagement renders the role management page.
- Roles: admin */
func (c Admin) RoleManagement() revel.Result {

	c.Log.Debug("render role management page", "url", c.Request.URL)
	c.Session["callPath"] = c.Request.URL.String()
	c.ViewArgs["tabName"] = c.Message("admin.tab")

	//get all admins
	var admins models.Users
	if err := admins.Get(models.ADMIN); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	//get all creators
	var creators models.Users
	if err := creators.Get(models.CREATOR); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	return c.Render(admins, creators)
}

/*Dashboard renders the dashboard.
- Roles: admin */
func (c Admin) Dashboard() revel.Result {

	c.Log.Debug("render dashboard", "url", c.Request.URL)
	c.Session["callPath"] = c.Request.URL.String()
	c.ViewArgs["tabName"] = c.Message("admin.tab")

	return c.Render()
}

/*SearchUser renders search results for a search value.
- Roles: admin */
func (c Admin) SearchUser(value string, searchInactive bool) revel.Result {

	c.Log.Debug("search users", "value", value, "searchInactive", searchInactive)

	trimmedValue := strings.TrimSpace(value)
	c.Validation.MinSize(trimmedValue, 3).MessageKey("validation.invalid.searchValue")
	c.Validation.MaxSize(trimmedValue, 127).MessageKey("validation.invalid.searchValue")
	if c.Validation.HasErrors() {
		c.Validation.Keep()
		return c.Render()
	}

	var users models.Users
	if err := users.Search(&value, &searchInactive); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}
	return c.Render(users)
}

/*UserDetails renders detailed information about an user.
- Roles: admin */
func (c Admin) UserDetails(ID int) revel.Result {

	c.Log.Debug("get user details", "userID", ID)

	c.Validation.Required(ID)
	if c.Validation.HasErrors() {
		return renderError(
			errors.New("missing user ID"),
			c.Controller,
		)
	}

	user := models.UserDetails{User: models.User{ID: ID}}
	if err := user.Get(); err != nil {
		return renderError(
			err,
			c.Controller,
		)
	}

	return c.Render(user)
}

/*ChangeRole changes the role of an user and sends a notification e-mail.
- Roles: admin */
func (c Admin) ChangeRole(user models.User) revel.Result {

	c.Log.Debug("change user role", "userID", user.ID, "role", user.Role)

	c.Validation.Required(user.ID).MessageKey("validation.missing.userID")
	if user.Role != models.ADMIN && user.Role != models.CREATOR &&
		user.Role != models.USER {
		c.Validation.ErrorKey("validation.invalid.role")
	}
	if c.Validation.HasErrors() {
		return flashError(
			errValidation,
			nil,
			c.Session["callPath"].(string),
			c.Controller,
			"",
		)
	}

	if err := user.ChangeRole(); err != nil {
		return flashError(
			errDB,
			err,
			c.Session["callPath"].(string),
			c.Controller,
			"",
		)
	}

	err := c.sendEMail(&user,
		"email.subject.new.role",
		"newRole")
	if err != nil {
		return flashError(
			errEMail,
			err,
			c.Session["callPath"].(string),
			c.Controller,
			user.EMail,
		)
	}

	c.Flash.Success(c.Message("admin.new.role.success",
		user.FirstName,
		user.LastName,
		c.Message("user.role."+user.Role.String()),
	))

	return c.Redirect(c.Session["callPath"].(string))
}

/*AddGroup adds a new group.
- Roles: admin */
func (c Admin) AddGroup(group models.Group) revel.Result {

	c.Log.Debug("add group", "group", group)

	if group.Validate(c.Validation); c.Validation.HasErrors() {
		return flashError(
			errValidation,
			nil,
			c.Session["callPath"].(string),
			c.Controller,
			"",
		)
	}

	userID := c.Session["userID"].(string)
	if err := group.Add(&userID); err != nil {
		return flashError(
			errDB,
			err,
			c.Session["callPath"].(string),
			c.Controller,
			"",
		)
	}

	c.Flash.Success(c.Message("group.new.success", group.Name, group.ID))
	return c.Redirect(c.Session["callPath"].(string))
}

/*EditGroup edits the name and the course limits of a group.
- Roles: admin */
func (c Admin) EditGroup(group models.Group) revel.Result {

	c.Log.Debug("edit group", "group", group)

	if group.Validate(c.Validation); c.Validation.HasErrors() {
		return flashError(
			errValidation,
			nil,
			c.Session["callPath"].(string),
			c.Controller,
			"",
		)
	}

	userID := c.Session["userID"].(string)
	if err := group.Edit(&userID); err != nil {
		return flashError(
			errDB,
			err,
			c.Session["callPath"].(string),
			c.Controller,
			"",
		)
	}

	c.Flash.Success(c.Message("group.edit.success", group.Name, group.ID))
	return c.Redirect(c.Session["callPath"].(string))
}

/*DeleteGroup deletes a group. Groups can only be deleted if it has no
sub groups and no active courses. Upon deletion, all inactive courses of
that group become the children of the parent group.
- Roles: admin */
func (c Admin) DeleteGroup(group models.Group) revel.Result {

	c.Log.Debug("delete group", "group", group)

	c.Validation.Check(group.ID,
		models.NoActiveChildren{},
		revel.Required{},
	).MessageKey("validation.invalid.groupID")
	//NOTE: the DB statement ensures that no courses are edited

	if c.Validation.HasErrors() {
		return flashError(
			errValidation,
			nil,
			c.Session["callPath"].(string),
			c.Controller,
			"",
		)
	}

	if err := group.Delete(); err != nil {
		return flashError(
			errDB,
			err,
			c.Session["callPath"].(string),
			c.Controller,
			"",
		)
	}

	c.Flash.Success(c.Message("group.delete.success", group.ID))
	return c.Redirect(c.Session["callPath"].(string))
}

//sendEMail sends an notification e-mail about a new user role.
func (c Admin) sendEMail(user *models.User, subjectKey string, filename string) (err error) {

	c.Log.Debug("sending EMail", "user", user, "subjectKey", subjectKey,
		"filename", filename)

	data := models.EMailData{User: *user}

	if !user.Language.Valid {
		user.Language.String = app.DefaultLanguage
	}

	email := app.EMail{
		Recipient: user.EMail,
	}

	err = models.GetEMailSubjectBody(
		&data,
		&user.Language.String,
		subjectKey,
		filename,
		&email,
		c.Controller,
	)
	if err != nil {
		return
	}

	c.Log.Debug("assembled e-mail", "email", email)

	app.EMailQueue <- email
	return
}
