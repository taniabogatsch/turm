package controllers

import (
	"turm/app/models"
	"turm/app/routes"

	"github.com/revel/revel"
)

/*AddGroup adds a new group.
- Roles: admin */
func (c Admin) AddGroup(group models.Group) revel.Result {

	c.Log.Debug("add group", "group", group)

	if group.Validate(c.Validation); c.Validation.HasErrors() {
		return flashError(
			errValidation,
			nil,
			routes.App.Index(),
			"",
			c.Controller,
			"",
		)
	}

	userID := c.Session["userID"].(string)
	if err := group.Add(&userID); err != nil {
		return flashError(
			errDB,
			err,
			routes.App.Index(),
			"",
			c.Controller,
			"",
		)
	}

	c.Flash.Success(c.Message("group.new.success", group.Name, group.ID))
	return c.Redirect(App.Index)
}

/*EditGroup edits the name and the course limits of a group.
- Roles: admin */
func (c Admin) EditGroup(group models.Group) revel.Result {

	c.Log.Debug("edit group", "group", group)

	if group.Validate(c.Validation); c.Validation.HasErrors() {
		return flashError(
			errValidation,
			nil,
			routes.App.Index(),
			"",
			c.Controller,
			"",
		)
	}

	userID := c.Session["userID"].(string)
	if err := group.Edit(&userID); err != nil {
		return flashError(
			errDB,
			err,
			routes.App.Index(),
			"",
			c.Controller,
			"",
		)
	}

	c.Flash.Success(c.Message("group.edit.success", group.Name, group.ID))
	return c.Redirect(App.Index)
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
			routes.App.Index(),
			"",
			c.Controller,
			"",
		)
	}

	if err := group.Delete(); err != nil {
		return flashError(
			errDB,
			err,
			routes.App.Index(),
			"",
			c.Controller,
			"",
		)
	}

	c.Flash.Success(c.Message("group.delete.success", group.ID))
	return c.Redirect(App.Index)
}
