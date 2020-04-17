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

	c.Validation.Check(language, models.LanguageValidator{})
	if c.Validation.HasErrors() {
		return flashError(
			errValidation,
			c.Session["callPath"].(string),
			"validation.invalid.language",
			c.Controller,
		)
	}

	c.Session["currentLocale"] = language
	c.ViewArgs["currentLocale"] = c.Session["currentLocale"]

	//some pages do not change the callPath and must be detected manually
	switch c.Session["currPath"] {
	case routes.User.LoginPage():
		return c.Redirect(User.LoginPage)
	case routes.User.RegistrationPage():
		return c.Redirect(User.RegistrationPage)
	case routes.User.NewPasswordPage():
		return c.Redirect(User.NewPasswordPage)
	case routes.User.ActivationPage():
		return c.Redirect(User.ActivationPage)
	case routes.User.PrefLanguagePage():
		return c.Redirect(User.PrefLanguagePage)
	}

	return c.Redirect(c.Session["callPath"])
}
