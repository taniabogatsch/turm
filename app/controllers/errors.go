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
func flashError(errType ErrorType, err error, url string, msg string, c *revel.Controller, i interface{}) revel.Result {

	//TODO: this will later allow to send an e-mail if any error occurs

	c.FlashParams()
	if err != nil {
		c.Log.Error(err.Error())
	}
	c.Log.Warn(errType.String(), "redirect", url)

	switch errType {

	case errValidation:
		c.Validation.Keep()
		return c.Redirect(url)

	case errDB:
		c.Flash.Error(c.Message("error.db"))
		return c.Redirect(url)

	case errAuth:
		c.Validation.Keep()
		return c.Redirect(url)

	case errEMail:
		email, parsed := i.(string)
		if !parsed {
			c.Log.Error("error parsing e-mail", "email", email)
		}
		c.Flash.Error(c.Message("error.email", email))
		return c.Redirect(url)

	case errTypeConv:
		c.Flash.Error(c.Message("error.typeConv"))
		return c.Redirect(url)

	default:
		c.Log.Error("undefined error type", "error type", errType)
		return c.Redirect(App.Index)
	}
}

//renderError renders a template containing the error.
func renderError(errType ErrorType, err error, msg string, c *revel.Controller, i interface{}) revel.Result {

	//TODO: this will later allow to send an e-mail if any error occurs

	if err != nil {
		c.Log.Error(err.Error())
	}
	templatePath := ""

	switch errType {

	case errContent:
		msg = c.Message("error.content")

	default:
		c.Log.Error("undefined error type")
		msg = c.Message("error.undefined")
	}

	c.ViewArgs["msg"] = msg
	templatePath = "errors/render.html"

	c.Log.Warn(errType.String(), "render", templatePath)
	return c.RenderTemplate(templatePath)
}
