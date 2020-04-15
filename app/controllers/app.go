package controllers

import (
	"turm/app/models"

	"github.com/revel/revel"
)

/*Index renders the landing page of the application.
- Roles: all (except not activated users) */
func (c App) Index() revel.Result {

	revel.AppLog.Debug("requesting index page")
	c.Session["callPath"] = "/"
	c.Session["currPath"] = "/"
	c.ViewArgs["tabName"] = c.Message("index.tabName")

	//TODO: get last update

	return c.Render()
}

/*ChangeLanguage changes the language, then redirects to the page currently set as callPath.
- Roles: all */
func (c App) ChangeLanguage(language string) revel.Result {

	revel.AppLog.Debug("change language", "language", language)

	if c.Validation.Check(language, models.LanguageValidator{}); c.Validation.HasErrors() {
		return flashError(errValidation, c.Controller, c.Session["callPath"].(string), "validation.invalid.language")
	}

	c.Session["currentLocale"] = language
	c.ViewArgs["currentLocale"] = c.Session["currentLocale"]

	//some pages do not change the callPath and must be detected manually
	switch c.Session["currPath"] {
	case c.Message("login.tabName"):
		return c.Redirect(User.LoginPage)
	case c.Message("register.tabName"):
		return c.Redirect(User.RegistrationPage)
	case c.Message("newPw.tabName"):
		return c.Redirect(User.NewPasswordPage)
	}

	return c.Redirect(c.Session["callPath"])
}
