package controllers

import (
	"turm/app/models"

	"github.com/revel/revel"
)

/*Index of the administration page.
- Roles: admin (activated) */
func (c Admin) Index() revel.Result {

	c.Log.Debug("render admin index page", "url", c.Request.URL)

	c.Session["callPath"] = c.Request.URL.String()
	c.Session["currPath"] = c.Request.URL.String()
	c.Session["lastURL"] = c.Request.URL.String()

	c.ViewArgs["tab"] = c.Message("admin.tab")

	return c.Render()
}

/*Dashboard rendering. The dashboard shows page statistics.
- Roles: admin (activated) */
func (c Admin) Dashboard() revel.Result {

	c.Log.Debug("render dashboard")
	c.Session["lastURL"] = c.Request.URL.String()

	return c.Render()
}

/*LogEntries rendering. Loads all not yet solved log entries of the error log.
- Roles: admin (activated) */
func (c Admin) LogEntries() revel.Result {

	c.Log.Debug("render log entries")
	c.Session["lastURL"] = c.Request.URL.String()

	var logEntries models.LogEntries
	if err := logEntries.Select(); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	return c.Render(logEntries)
}

/*InsertLogEntries inserts all new log entries of the error log file. These are all
log entries appended after the last executing of this controller.
- Roles: admin (activated) */
func (c Admin) InsertLogEntries() revel.Result {

	c.Log.Debug("insert new log entries")
	c.Session["lastURL"] = c.Request.URL.String()

	var logEntries models.LogEntries
	if err := logEntries.Insert(); err != nil {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message(errDB.String())})
	}

	return c.RenderJSON(
		response{Status: SUCCESS, Msg: c.Message("admin.insert.log.entries.success")})
}

/*SolveLogEntry solves a log entry. Any log entry being solved is no longer shown
when rendering all log entries.
- Roles: admin (activated) */
func (c Admin) SolveLogEntry(entry models.LogEntry) revel.Result {

	c.Log.Debug("solve log entry", "entry", entry)
	c.Session["lastURL"] = c.Request.URL.String()

	if err := entry.Solve(); err != nil {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message(errDB.String())})
	}

	return c.RenderJSON(
		response{Status: SUCCESS, Msg: c.Message("admin.solve.log.entry.success")})
}

/*Users rendering. Loads a modal for loading user details. Also loads the details
of the provided user (if ID != 0).
- Roles: admin (activated) */
func (c Admin) Users(ID int) revel.Result {

	c.Log.Debug("render user details", "ID", ID)
	c.Session["lastURL"] = c.Request.URL.String()

	if ID != 0 {
		user := models.UserDetails{User: models.User{ID: ID}}
		if err := user.Get(); err != nil {
			return renderError(err, c.Controller)
		}
		return c.Render(user)
	}

	return c.Render()
}

/*SearchUser renders search results for a search value. The search results are a
slice of users.
- Roles: admin (activated) */
func (c Admin) SearchUser(value string, searchInactive bool) revel.Result {

	c.Log.Debug("search users", "value", value, "searchInactive", searchInactive)
	c.Session["lastURL"] = c.Request.URL.String()

	models.ValidateLength(&value, "validation.invalid.searchValue",
		3, 127, c.Validation)

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

/*ChangeRole of an user. Sends a notification e-mail on success.
- Roles: admin (activated) */
func (c Admin) ChangeRole(user models.User) revel.Result {

	c.Log.Debug("change user role", "user", user)
	c.Session["lastURL"] = c.Request.URL.String()

	c.Validation.Required(user.ID).
		MessageKey("validation.missing.userID")
	if user.Role < models.USER || user.Role > models.ADMIN {
		c.Validation.ErrorKey("validation.invalid.role")
	}

	if c.Validation.HasErrors() {
		return c.RenderJSON(
			response{Status: INVALID, Msg: getErrorString(c.Validation.Errors)})
	}

	if err := user.ChangeRole(); err != nil {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message(errDB.String())})
	}

	data := models.EMailData{User: user}
	err := sendEMail(c.Controller, &data,
		"email.subject.new.role",
		"newRole")

	if err != nil {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message(errEMail.String())})
		//TODO: use e-mail in wrapper
		//return flashError(errEMail, err, "", c.Controller, user.EMail)
	}

	//update the session if the user updated his own role
	sessionID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message(errTypeConv.String())})
	}
	if sessionID == user.ID {
		c.Session["role"] = user.Role.String()
	}

	return c.RenderJSON(
		response{Status: SUCCESS, Msg: c.Message("admin.new.role.success",
			user.FirstName,
			user.LastName,
			c.Message("user.role."+user.Role.String()),
		)})
}

/*ChangeUserData updates the salutation, first name, last name and e-mail of
an user.
- Roles: admin (activated) */
func (c Admin) ChangeUserData(user models.User) revel.Result {

	c.Log.Debug("change user data", "user", user)
	c.Session["lastURL"] = c.Request.URL.String()

	if err := user.ChangeUserData(c.Validation); err != nil {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message(errDB.String())})
	} else if c.Validation.HasErrors() {
		return c.RenderJSON(
			response{Status: INVALID, Msg: getErrorString(c.Validation.Errors)})
	}

	//update the session if the user updated his own data
	sessionID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		return c.RenderJSON(
			response{Status: ERROR, Msg: c.Message(errTypeConv.String())})
	}
	if sessionID == user.ID {
		c.Session["firstName"] = user.FirstName
		c.Session["lastName"] = user.LastName
		c.Session["eMail"] = user.EMail
	}

	return c.RenderJSON(
		response{Status: SUCCESS, Msg: c.Message("admin.change.data.success",
			user.FirstName,
			user.LastName,
		)})
}

/*Roles renders all users with elevated roles (ADMIN or CREATOR).
- Roles: admin (activated) */
func (c Admin) Roles() revel.Result {

	c.Log.Debug("render users with elevated roles")
	c.Session["lastURL"] = c.Request.URL.String()

	//get all admins and creators
	var admins models.Users
	var creators models.Users

	if err := admins.Get(&creators); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	return c.Render(admins, creators)
}

/*InsertGroup inserts a new group. A group can be the child of another group or
a root group. Each active course must be part of a group.
- Roles: admin (activated) */
func (c Admin) InsertGroup(group models.Group) revel.Result {

	c.Log.Debug("insert group", "group", group)
	c.Session["lastURL"] = c.Request.URL.String()

	if group.Validate(c.Validation); c.Validation.HasErrors() {
		return flashError(errValidation, nil, "", c.Controller, "")
	}

	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		return flashError(errTypeConv, err, "", c.Controller, "")
	}

	if err = group.Insert(&userID); err != nil {
		return flashError(errDB, err, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("group.insert.success",
		group.Name,
		group.ID,
	))
	return c.Redirect(c.Session["currPath"])
}

/*UpdateGroup updates the name and the course limit of a group.
- Roles: admin (activated) */
func (c Admin) UpdateGroup(group models.Group) revel.Result {

	c.Log.Debug("update group", "group", group)
	c.Session["lastURL"] = c.Request.URL.String()

	c.Validation.Required(group.ID).
		MessageKey("validation.invalid.params")

	if group.Validate(c.Validation); c.Validation.HasErrors() {
		return flashError(errValidation, nil, "", c.Controller, "")
	}

	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		return flashError(errTypeConv, err, "", c.Controller, "")
	}

	if err = group.Update(&userID); err != nil {
		return flashError(errDB, err, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("group.update.success",
		group.Name,
		group.ID,
	))
	return c.Redirect(c.Session["currPath"])
}

/*DeleteGroup deletes a group. Groups can only be deleted if they have no children and contain
no active courses. Upon deletion, all inactive courses of that group become the
children of the parent group.
- Roles: admin (activated) */
func (c Admin) DeleteGroup(ID int) revel.Result {

	c.Log.Debug("delete group", "ID", ID)
	c.Session["lastURL"] = c.Request.URL.String()

	c.Validation.Check(ID,
		models.NoActiveChildren{},
		revel.Required{},
	).MessageKey("validation.invalid.groupID")

	if c.Validation.HasErrors() {
		return flashError(errValidation, nil, "", c.Controller, "")
	}

	group := models.Group{ID: ID}
	if err := group.Delete(); err != nil {
		return flashError(errDB, err, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("group.delete.success", group.ID))
	return c.Redirect(c.Session["currPath"])
}

/*InsertCategory inserts the new category as a FAQ category or a news feed
category.
- Roles: admin (activated) */
func (c Admin) InsertCategory(category models.Category, table string) revel.Result {

	c.Log.Debug("insert category", "category", category, "table", table)
	c.Session["lastURL"] = c.Request.URL.String()

	if table != tabFAQCategory && table != tabNewsFeedCategory {
		c.Validation.ErrorKey("validation.invalid.params")
	}

	if category.Validate(c.Validation); c.Validation.HasErrors() {
		return flashError(errValidation, nil, "", c.Controller, "")
	}

	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		return flashError(errTypeConv, err, "", c.Controller, "")
	}

	if err = category.Insert(&table, &userID); err != nil {
		return flashError(errDB, err, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("category.insert.success",
		category.Name,
		category.ID,
	))
	return c.Redirect(c.Session["currPath"])
}

/*UpdateCategory updates the name of a category.
- Roles: admin (activated) */
func (c Admin) UpdateCategory(category models.Category, table string) revel.Result {

	c.Log.Debug("update category", "category", category, "table", table)
	c.Session["lastURL"] = c.Request.URL.String()

	if table != tabFAQCategory && table != tabNewsFeedCategory {
		c.Validation.ErrorKey("validation.invalid.params")
	}

	c.Validation.Required(category.ID).
		MessageKey("validation.invalid.params")

	if category.Validate(c.Validation); c.Validation.HasErrors() {
		return flashError(errValidation, nil, "", c.Controller, "")
	}

	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		return flashError(errTypeConv, err, "", c.Controller, "")
	}

	if err := category.Update(&table, &userID); err != nil {
		return flashError(errDB, err, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("category.update.success",
		category.Name,
		category.ID,
	))
	return c.Redirect(c.Session["currPath"])
}

/*DeleteCategory deletes a category.
- Roles: admin (activated) */
func (c Admin) DeleteCategory(ID int, table string) revel.Result {

	c.Log.Debug("delete category", "ID", ID, "table", table)
	c.Session["lastURL"] = c.Request.URL.String()

	if table != tabFAQCategory && table != tabNewsFeedCategory {
		c.Validation.ErrorKey("validation.invalid.params")
	}

	c.Validation.Required(ID).
		MessageKey("validation.invalid.params")

	if c.Validation.HasErrors() {
		return flashError(errValidation, nil, "", c.Controller, "")
	}

	category := models.Category{ID: ID}
	if err := category.Delete(&table); err != nil {
		return flashError(errDB, err, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("category.delete.success", category.ID))
	return c.Redirect(c.Session["currPath"])
}

/*InsertHelpPageEntry inserts a new entry, which is either a FAQ or a news feed
entry.
- Roles: admin (activated) */
func (c Admin) InsertHelpPageEntry(entry models.HelpPageEntry) revel.Result {

	c.Log.Debug("insert help page entry", "entry", entry)
	c.Session["lastURL"] = c.Request.URL.String()

	if entry.Validate(c.Validation); c.Validation.HasErrors() {
		return flashError(errValidation, nil, "", c.Controller, "")
	}

	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		return flashError(errTypeConv, err, "", c.Controller, "")
	}

	if err := entry.Insert(&userID); err != nil {
		return flashError(errDB, err, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("entry.insert.success",
		entry.CategoryID,
		entry.ID,
	))
	return c.Redirect(c.Session["currPath"])
}

/*UpdateHelpPageEntry updates a FAQ or news feed entry.
- Roles: admin (activated) */
func (c Admin) UpdateHelpPageEntry(entry models.HelpPageEntry) revel.Result {

	c.Log.Debug("update help page entry", "entry", entry)
	c.Session["lastURL"] = c.Request.URL.String()

	c.Validation.Required(entry.ID).
		MessageKey("validation.invalid.params")

	if entry.Validate(c.Validation); c.Validation.HasErrors() {
		return flashError(errValidation, nil, "", c.Controller, "")
	}

	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		return flashError(errTypeConv, err, "", c.Controller, "")
	}

	if err := entry.Update(&userID); err != nil {
		return flashError(errDB, err, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("entry.update.success",
		entry.CategoryID,
		entry.ID,
	))
	return c.Redirect(c.Session["currPath"])
}

/*DeleteHelpPageEntry deletes a FAQ or news feed entry.
- Roles: admin (activated) */
func (c Admin) DeleteHelpPageEntry(ID int, table string) revel.Result {

	c.Log.Debug("delete help page entry", "ID", ID, "table", table)
	c.Session["lastURL"] = c.Request.URL.String()

	if table != "faqs" && table != "news_feed" {
		c.Validation.ErrorKey("validation.invalid.params")
	}

	c.Validation.Required(ID).
		MessageKey("validation.invalid.params")

	if c.Validation.HasErrors() {
		return flashError(errValidation, nil, "", c.Controller, "")
	}

	entry := models.HelpPageEntry{ID: ID}
	if err := entry.Delete(&table); err != nil {
		return flashError(errDB, err, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("entry.delete.success", entry.ID))
	return c.Redirect(c.Session["currPath"])
}
