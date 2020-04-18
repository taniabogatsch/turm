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
	errDataConv
)

func (s ErrorType) String() string {
	return [...]string{"validation failed", "database error",
		"authentication failed", "e-mail error", "data conversion error"}[s]
}

//flashError flashes an error message and redirects to a page.
func flashError(errType ErrorType, err error, url string, msg string, c *revel.Controller, i interface{}) revel.Result {

	//TODO: this will later allow to send an e-mail if any error occurs

	c.FlashParams()
	if err != nil {
		c.Log.Error(err.Error())
	}
	c.Log.Warn(errType.String(), "redirect", url)

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
		//keep the validation errors and flash the parameters, then redirect
		c.Validation.Keep()
		return c.Redirect(url)

	case errEMail:
		//flash error and parameters, then redirect
		email, parsed := i.(string)
		if !parsed {
			c.Log.Error("error parsing e-mail", "email", email)
		}
		c.Flash.Error(c.Message("error.email", email))
		return c.Redirect(url)

	case errDataConv:
		//flash error and parameters, then redirect
		c.Flash.Error(c.Message("error.typeConversion"))
		return c.Redirect(url)

	default:
		c.Log.Error("undefined error type")
		return c.Redirect(App.Index)
	}
}
