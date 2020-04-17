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
	errEMail
)

func (s ErrorType) String() string {
	return [...]string{"validation error", "database error", "authentification error"}[s]
}

//flashError flashes an error message and redirects to a page.
func flashError(errType ErrorType, url string, msg string, c *revel.Controller) revel.Result {

	//TODO: this will later allow to send an e-mail if any error occurs

	c.FlashParams()

	switch errType {

	case errValidation:
		//keep the validation errors and flash the parameters, then redirect
		c.Validation.Keep()
		return c.Redirect(url)

	case errDB:
		//flash error and parameters, then redirect
		c.Flash.Error(c.Message("error.database"))
		return c.Redirect(url)

	case errAuth:
		//flash error and parameters, then redirect
		c.Flash.Error(c.Message(msg))
		return c.Redirect(url)

	case errEMail:
		//flash error and parameters, then redirect
		c.Flash.Error(c.Message("error.email"))
		return c.Redirect(url)

	default:
		return c.Redirect(App.Index)
	}
}
