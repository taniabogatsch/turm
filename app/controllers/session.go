package controllers

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
