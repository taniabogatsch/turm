package controllers

import (
	"errors"
	"turm/app/models"

	"github.com/revel/revel"
)

/*Index renders the landing page of the application.
- Roles: all (except not activated users) */
func (c App) Index() revel.Result {

	c.Log.Debug("render index page", "url", c.Request.URL)
	c.Session["callPath"] = c.Request.URL.String()
	c.Session["currPath"] = c.Request.URL.String()
	c.ViewArgs["tabName"] = c.Message("index.tab")

	return c.Render()
}

/*Groups renders all groups.
- Roles: all (except not activated users) */
func (c App) Groups(prefix string) revel.Result {

	c.Log.Debug("get groups", "prefix", prefix)

	c.Validation.Required(prefix)
	if c.Validation.HasErrors() {
		return renderError(
			errors.New("missing prefix"),
			c.Controller,
		)
	}

	var Groups models.Groups
	if err := Groups.Get(&prefix); err != nil {
		return renderError(
			err,
			c.Controller,
		)
	}

	return c.Render(Groups)
}

/*ChangeLanguage changes the language, then redirects to the page currently set as currPath.
- Roles: all */
func (c App) ChangeLanguage(language string) revel.Result {

	c.Log.Debug("change language",
		"old language", c.Session["currentLocale"],
		"language", language)

	c.Validation.Check(language,
		models.LanguageValidator{},
	).MessageKey("validation.invalid.language")

	if c.Validation.HasErrors() {
		return flashError(
			errValidation, nil, "", c.Controller, "")
	}

	c.Session["currentLocale"] = language
	c.ViewArgs["currentLocale"] = c.Session["currentLocale"]
	c.Request.Locale = c.Session["currentLocale"].(string)

	c.Flash.Success(c.Message("language.change.success",
		language,
	))
	return c.Redirect(c.Session["currPath"])
}
