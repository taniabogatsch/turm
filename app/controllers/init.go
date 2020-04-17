package controllers

import (
	"turm/app"
	"turm/app/routes"

	"github.com/revel/revel"
)

//init initializes all interceptors.
func init() {

	//initialize general interceptor
	revel.InterceptFunc(general, revel.BEFORE, &revel.Controller{})
}

//general intercepts each revel controller.
//It sets the service e-mail, the languages, the current language,
//the call path (if not set) and resets the logout timer.
func general(c *revel.Controller) revel.Result {

	c.ViewArgs["serviceEMail"] = app.ServiceEMail
	c.ViewArgs["languages"] = app.Languages

	//set language
	if c.Session["currentLocale"] == nil {
		//set default language as set in the config file
		c.Session.Set("currentLocale", app.DefaultLanguage)
	}
	c.ViewArgs["currentLocale"] = c.Session["currentLocale"]

	//ensure that callPath and currPath is not nil
	if c.Session["callPath"] == nil {
		c.Session["callPath"] = routes.App.Index()
	}
	if c.Session["currPath"] == nil {
		c.Session["currPath"] = routes.App.Index()
	}

	//reset logout timer
	c.Session.SetDefaultExpiration()
	if c.Session["stayLoggedIn"] != nil {
		if c.Session["stayLoggedIn"] == "true" {
			c.Session.SetNoExpiration()
		}
	}

	return nil
}