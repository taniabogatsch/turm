package controllers

import (
	"turm/app"

	"github.com/revel/revel"
)

//init initializes all interceptors.
func init() {

	//initialize general interceptor
	revel.InterceptFunc(general, revel.BEFORE, &revel.Controller{})
}

//general ... TODO.
func general(c *revel.Controller) revel.Result {

	c.ViewArgs["serviceMail"] = app.ServiceMail
	c.ViewArgs["languages"] = app.Languages

	//set language
	if c.Session["currentLocale"] == nil {
		//set default language as set in the config file
		c.Session.Set("currentLocale", app.DefaultLanguage)
	}
	c.ViewArgs["currentLocale"] = c.Session["currentLocale"]

	//ensure that callPath is not nil
	if c.Session["callPath"] == nil {
		c.Session["callPath"] = "/"
	}

	//reset logout timer
	if c.Session["stayLoggedIn"] != nil {
		if c.Session["stayLoggedIn"] == "true" {
			c.Session.SetNoExpiration()
		} else {
			c.Session.SetDefaultExpiration()
		}
	}

	return nil
}
