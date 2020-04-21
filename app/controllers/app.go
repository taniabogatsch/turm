package controllers

import (
	"errors"
	"turm/app/models"
	"turm/app/routes"

	"github.com/revel/revel"
)

/*Index renders the landing page of the application.
- Roles: all (except not activated users) */
func (c App) Index() revel.Result {

	c.Log.Debug("render index page", "url", c.Request.URL)
	c.Session["callPath"] = c.Request.URL.String()
	c.ViewArgs["tabName"] = c.Message("index.tab")

	//TODO: get last update

	return c.Render()
}

/*Groups renders all groups.
- Roles: all */
func (c App) Groups(prefix string) revel.Result {

	c.Log.Debug("get groups", "prefix", prefix)

	c.Validation.Required(prefix)
	if c.Validation.HasErrors() {
		return renderError(
			errContent,
			errors.New("missing prefix"),
			"",
			c.Controller,
			"",
		)
	}

	var Groups models.Groups
	if err := Groups.Get(&prefix); err != nil {
		return renderError(
			errDB,
			err,
			"",
			c.Controller,
			"",
		)
	}

	return c.Render(Groups)
}

/*ChangeLanguage changes the language, then redirects to the page currently set as callPath.
- Roles: all */
func (c App) ChangeLanguage(language string) revel.Result {

	c.Log.Debug("change language",
		"old language", c.Session["currentLocale"],
		"language", language)

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
			nil,
			c.Session["callPath"].(string),
			"",
			c.Controller,
			"",
		)
	}

	c.Session["currentLocale"] = language
	c.ViewArgs["currentLocale"] = c.Session["currentLocale"]
	c.Request.Locale = c.Session["currentLocale"].(string)

	c.Flash.Success(c.Message("language.change.success", language))
	return c.Redirect(c.Session["callPath"])
}
