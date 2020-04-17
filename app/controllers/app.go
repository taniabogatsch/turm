package controllers

import (
	"turm/app/models"
	"turm/app/routes"

	"github.com/revel/revel"
)

/*Index renders the landing page of the application.
- Roles: all (except not activated users) */
func (c App) Index() revel.Result {

	revel.AppLog.Debug("requesting index page")
	c.Session["callPath"] = c.Request.URL
	c.ViewArgs["tabName"] = c.Message("index.tabName")

	//TODO: get last update

	return c.Render()
}

/*ChangeLanguage changes the language, then redirects to the page currently set as callPath.
- Roles: all */
func (c App) ChangeLanguage(language string) revel.Result {

	revel.AppLog.Debug("change language", "language", language)

	//some pages do not change the callPath and must be detected manually
	if c.Session["currPath"] != routes.App.Index() {
		c.Session["callPath"] = c.Session["currPath"]
	}

	c.Validation.Check(language,
		models.LanguageValidator{},
	).MessageKey("validation.invalid.language")

	if c.Validation.HasErrors() {
		return flashError(
			errValidation,
			c.Session["callPath"].(string),
			"validation.invalid.language",
			c.Controller,
			"",
		)
	}

	c.Session["currentLocale"] = language
	c.ViewArgs["currentLocale"] = c.Session["currentLocale"]

	return c.Redirect(c.Session["callPath"])
}
