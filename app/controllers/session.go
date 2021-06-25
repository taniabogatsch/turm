package controllers

/*
Next to user related key-value pairs, the session contains the following three important key-value pairs.

currPath:
The path of the current page. Used for redirection after
- changing the language
- redirecting some admin controllers
- validating a course
- redirecting after enrollment

callPath:
The page calling the current page. Used for redirection after
- login
- verifying the activation code
- setting the preferred language

lastURL:
The url of the last controller being executed. This session value ensures easier debugging.
*/

import (
	"strconv"

	"github.com/revel/revel"
)

//getIntFromSession returns the int value from the session
func getIntFromSession(c *revel.Controller, key string) (value int, err error) {

	if c.Session[key] != nil {
		value, err = strconv.Atoi(c.Session[key].(string))
		if err != nil {
			c.Log.Error("failed to get "+key+" from session", "error", err.Error())
		}
	}
	return
}
