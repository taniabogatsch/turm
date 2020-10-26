package controllers

import (
	"strings"
	"turm/app/models"

	"github.com/revel/revel"
)

/*Users renders the user management page.
- Roles: admin (activated) */
func (c Admin) Users() revel.Result {

	c.Log.Debug("render user management page", "url", c.Request.URL)
	c.Session["callPath"] = c.Request.URL.String()
	c.Session["currPath"] = c.Request.URL.String()
	c.ViewArgs["tabName"] = c.Message("admin.tab")

	return c.Render()
}

/*Roles renders the role management page.
- Roles: admin (activated) */
func (c Admin) Roles() revel.Result {

	c.Log.Debug("render role management page", "url", c.Request.URL)
	c.Session["callPath"] = c.Request.URL.String()
	c.Session["currPath"] = c.Request.URL.String()
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
- Roles: admin (activated) */
func (c Admin) Dashboard() revel.Result {

	c.Log.Debug("render dashboard", "url", c.Request.URL)
	c.Session["callPath"] = c.Request.URL.String()
	c.Session["currPath"] = c.Request.URL.String()
	c.ViewArgs["tabName"] = c.Message("admin.tab")

	return c.Render()
}

/*SearchUser renders search results for a search value.
- Roles: admin (activated) */
func (c Admin) SearchUser(value string, searchInactive bool) revel.Result {

	c.Log.Debug("search users", "value", value, "searchInactive", searchInactive)

	value = strings.TrimSpace(value)
	c.Validation.Check(value,
		revel.MinSize{3},
		revel.MaxSize{127},
	).MessageKey("validation.invalid.searchValue")

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
- Roles: admin (activated) */
func (c Admin) UserDetails(ID int) revel.Result {

	c.Log.Debug("get user details", "userID", ID)

	//NOTE: no ID validation, if this controller is called with an
	//invalid ID, then something is going wrong

	user := models.UserDetails{User: models.User{ID: ID}}
	if err := user.Get(); err != nil {
		return renderError(err, c.Controller)
	}

	return c.Render(user)
}

/*ChangeRole changes the role of an user and sends a notification e-mail.
- Roles: admin (activated) */
func (c Admin) ChangeRole(user models.User) revel.Result {

	c.Log.Debug("change user role", "userID", user.ID, "role", user.Role)

	c.Validation.Required(user.ID).
		MessageKey("validation.missing.userID")
	if user.Role < models.USER || user.Role > models.ADMIN {
		c.Validation.ErrorKey("validation.invalid.role")
	}

	if c.Validation.HasErrors() {
		return flashError(
			errValidation, nil, "", c.Controller, "")
	}

	if err := user.ChangeRole(); err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
	}

	data := models.EMailData{User: user}
	err := sendEMail(c.Controller, &data,
		"email.subject.new.role",
		"newRole")
	if err != nil {
		return flashError(
			errEMail, err, "", c.Controller, user.EMail)
	}

	//update the session if the user updated his own role
	sessionID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		return flashError(
			errTypeConv, err, "", c.Controller, "")
	}
	if sessionID == user.ID {
		c.Session["role"] = user.Role.String()
	}

	c.Flash.Success(c.Message("admin.new.role.success",
		user.FirstName,
		user.LastName,
		c.Message("user.role."+user.Role.String()),
	))
	return c.Redirect(c.Session["currPath"])
}

/*InsertGroup inserts a new group.
- Roles: admin (activated) */
func (c Admin) InsertGroup(group models.Group) revel.Result {

	c.Log.Debug("insert group", "group", group)

	if group.Validate(c.Validation); c.Validation.HasErrors() {
		return flashError(
			errValidation, nil, "", c.Controller, "")
	}

	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		return flashError(
			errTypeConv, err, "", c.Controller, "")
	}

	if err = group.Insert(&userID); err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
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

	c.Validation.Required(group.ID).
		MessageKey("validation.invalid.params")

	if group.Validate(c.Validation); c.Validation.HasErrors() {
		return flashError(
			errValidation, nil, "", c.Controller, "")
	}

	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		return flashError(
			errTypeConv, err, "", c.Controller, "")
	}

	if err = group.Update(&userID); err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("group.update.success",
		group.Name,
		group.ID,
	))
	return c.Redirect(c.Session["currPath"])
}

/*DeleteGroup deletes a group. Groups can only be deleted if it has no
sub groups and no active courses. Upon deletion, all inactive courses of
that group become the children of the parent group.
- Roles: admin (activated) */
func (c Admin) DeleteGroup(ID int) revel.Result {

	c.Log.Debug("delete group", "ID", ID)

	c.Validation.Check(ID,
		models.NoActiveChildren{},
		revel.Required{},
	).MessageKey("validation.invalid.groupID")

	if c.Validation.HasErrors() {
		return flashError(
			errValidation, nil, "", c.Controller, "")
	}

	group := models.Group{ID: ID}
	if err := group.Delete(); err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("group.delete.success",
		group.ID,
	))
	return c.Redirect(c.Session["currPath"])
}

/*InsertCategory inserts a new category, either in the faq_category or the
news_feed_category table.
- Roles: admin (activated) */
func (c Admin) InsertCategory(category models.Category, table string) revel.Result {

	c.Log.Debug("insert category", "category", category, "table", table)

	if table != "faq_category" && table != "news_feed_category" {
		c.Validation.ErrorKey("validation.invalid.params")
	}

	if category.Validate(c.Validation); c.Validation.HasErrors() {
		return flashError(
			errValidation, nil, "", c.Controller, "")
	}

	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		return flashError(
			errTypeConv, err, "", c.Controller, "")
	}

	if err = category.Insert(&table, &userID); err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
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

	if table != "faq_category" && table != "news_feed_category" {
		c.Validation.ErrorKey("validation.invalid.params")
	}

	c.Validation.Required(category.ID).
		MessageKey("validation.invalid.params")

	if category.Validate(c.Validation); c.Validation.HasErrors() {
		return flashError(
			errValidation, nil, "", c.Controller, "")
	}

	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		return flashError(
			errTypeConv, err, "", c.Controller, "")
	}

	if err := category.Update(&table, &userID); err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
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

	if table != "faq_category" && table != "news_feed_category" {
		c.Validation.ErrorKey("validation.invalid.params")
	}

	c.Validation.Required(ID).
		MessageKey("validation.invalid.params")

	if c.Validation.HasErrors() {
		return flashError(
			errValidation, nil, "", c.Controller, "")
	}

	category := models.Category{ID: ID}
	if err := category.Delete(&table); err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("category.delete.success",
		category.ID,
	))
	return c.Redirect(c.Session["currPath"])
}

/*InsertHelpPageEntry inserts a new entry, either in the faq or the news_feed table.
- Roles: admin (activated) */
func (c Admin) InsertHelpPageEntry(entry models.HelpPageEntry) revel.Result {

	c.Log.Debug("insert entry", "entry", entry)

	if entry.Validate(c.Validation); c.Validation.HasErrors() {
		return flashError(
			errValidation, nil, "", c.Controller, "")
	}

	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		return flashError(
			errTypeConv, err, "", c.Controller, "")
	}

	if err := entry.Insert(&userID); err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("entry.insert.success",
		entry.CategoryID,
		entry.ID,
	))
	return c.Redirect(c.Session["currPath"])
}

/*UpdateHelpPageEntry updates an entry in either the faq (question/answer)
or the news_feed (content) table.
- Roles: admin (activated) */
func (c Admin) UpdateHelpPageEntry(entry models.HelpPageEntry) revel.Result {

	c.Log.Debug("update entry", "entry", entry)

	c.Validation.Required(entry.ID).
		MessageKey("validation.invalid.params")

	if entry.Validate(c.Validation); c.Validation.HasErrors() {
		return flashError(
			errValidation, nil, "", c.Controller, "")
	}

	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		return flashError(
			errTypeConv, err, "", c.Controller, "")
	}

	if err := entry.Update(&userID); err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("entry.update.success",
		entry.CategoryID,
		entry.ID,
	))
	return c.Redirect(c.Session["currPath"])
}

/*DeleteHelpPageEntry deletes an entry in either the faq or the news_feed table.
- Roles: admin (activated) */
func (c Admin) DeleteHelpPageEntry(ID int, table string) revel.Result {

	c.Log.Debug("delete entry", "ID", ID, "table", table)

	if table != "faqs" && table != "news_feed" {
		c.Validation.ErrorKey("validation.invalid.params")
	}

	c.Validation.Required(ID).
		MessageKey("validation.invalid.params")

	if c.Validation.HasErrors() {
		return flashError(
			errValidation, nil, "", c.Controller, "")
	}

	entry := models.HelpPageEntry{ID: ID}
	if err := entry.Delete(&table); err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("entry.delete.success",
		entry.ID,
	))
	return c.Redirect(c.Session["currPath"])
}
