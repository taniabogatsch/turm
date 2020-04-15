package controllers

import (
	"github.com/revel/revel"
)

/*ErrorType is a type for encoding different errors. */
type ErrorType int

const (
	errValidation ErrorType = iota
	errDB
	errAuth
)

func (s ErrorType) String() string {
	return [...]string{"validation error", "database error", "authentification error"}[s]
}

//flashError flashes an error message and redirects to a page.
func flashError(errType ErrorType, c *revel.Controller, url string, msg string) revel.Result {

	switch errType {

	case errValidation:
		//keep the validation errors and flash the parameters, then redirect
		c.Validation.Keep()
		c.FlashParams()
		return c.Redirect(url)

	case errDB:
		//flash error and parameters, then redirect
		c.Flash.Error(c.Message("error.database"))
		c.FlashParams()
		return c.Redirect(url)

	case errAuth:
		//flash error and parameters, then redirect
		c.Flash.Error(c.Message(msg))
		c.FlashParams()
		return c.Redirect(url)

	default:
		return c.Redirect(App.Index)
	}
}
