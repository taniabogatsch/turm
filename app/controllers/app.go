package controllers

import (
	"turm/app/models"

	"github.com/revel/revel"
)

/*App implements logic to CRUD general page data. */
type App struct {
	*revel.Controller
}

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
		c.Flash.Error(c.Message("validation.invalid.language"))
		return c.Redirect(c.Session["callPath"])
	}

	c.Session["currentLocale"] = language
	c.ViewArgs["currentLocale"] = c.Session["currentLocale"]

	//some pages do not change the callPath and must be detected manually
	if c.Session["currPath"] == c.Message("login.tabName") {
		return c.Redirect(User.LoginPage)
	}
	if c.Session["currPath"] == c.Message("register.tabName") {
		return c.Redirect(User.RegistrationPage)
	}
	/*
		if c.Session["currPath"] == c.Message("newPw.pageName") {
			return c.Redirect(App.NewPw)
		}
	*/
	return c.Redirect(c.Session["callPath"])
}
