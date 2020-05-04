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
	errTypeConv
	errContent
)

func (s ErrorType) String() string {
	return [...]string{"validation failed", "database error",
		"authentication failed", "e-mail error", "type conversion error",
		"error loading content"}[s]
}

//flashError flashes an error message and redirects to a page.
func flashError(errType ErrorType, err error, url string, c *revel.Controller, i interface{}) revel.Result {

	//TODO: this will later allow to send an e-mail if any error occurs

	c.FlashParams()
	if err != nil {
		c.Log.Error(err.Error())
	}
	c.Log.Warn(errType.String(), "redirect", url)

	switch errType {

	case errValidation, errAuth:
		c.Validation.Keep()

	case errDB:
		c.Flash.Error(c.Message("error.db"))

	case errEMail:
		email, parsed := i.(string)
		if !parsed {
			c.Log.Error("error parsing e-mail", "email", email)
		}
		c.Flash.Error(c.Message("error.email", email))

	case errTypeConv:
		c.Flash.Error(c.Message("error.typeConv"))

	default:
		c.Log.Error("undefined error type", "error type", errType)
		c.Flash.Error(c.Message("error.undefined"))
		return c.Redirect(App.Index)
	}

	return c.Redirect(url)
}

//renderError renders a template containing the error.
func renderError(err error, c *revel.Controller) revel.Result {

	//TODO: this will later allow to send an e-mail if any error occurs

	if err != nil {
		c.Log.Error(err.Error())
	}

	templatePath := "errors/render.html"

	c.ViewArgs["errMsg"] = c.Message("error.content")
	c.Validation.Keep()

	c.Log.Warn("render", "path", templatePath)
	return c.RenderTemplate(templatePath)
}

//renderQuietError renders an error message.
func renderQuietError(errType ErrorType, err error, c *revel.Controller) {

	//TODO: this will later allow to send an e-mail if any error occurs

	if err != nil {
		c.Log.Error(err.Error())
	}

	switch errType {
	case errDB:
		c.ViewArgs["errMsg"] = c.Message("error.db")
	default:
		c.ViewArgs["errMsg"] = c.Message("error.undefined")
	}
}
