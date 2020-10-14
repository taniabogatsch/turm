package controllers

import (
	"turm/app"

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
	return [...]string{"validation failed", "error.db",
		"error.auth", "e-mail error", "error.typeConv",
		"error loading content"}[s]
}

//flashError flashes an error message and redirects to a page.
func flashError(errType ErrorType, err error, url string, c *revel.Controller, i interface{}) revel.Result {

	c.FlashParams()
	if err != nil { //log error and send notification e-mail
		c.Log.Error("flash error", "err", err.Error())
		app.SendErrorNote()
	}

	if url == "" {
		url = c.Session["currPath"].(string)
	}
	c.Log.Debug(errType.String(), "redirect", url)

	//execute the correct error action
	switch errType {
	case errAuth, errDB, errTypeConv:
		c.Flash.Error(c.Message(errType.String()))
	case errValidation:
		c.Validation.Keep()
	case errEMail:
		email, parsed := i.(string)
		if !parsed {
			c.Log.Error("error parsing e-mail", "email", email)
		}
		c.Flash.Error(c.Message("error.email", email))
	default:
		c.Log.Error("undefined error type", "error type", errType)
		c.Flash.Error(c.Message("error.undefined"))
		return c.Redirect(App.Index)
	}

	return c.Redirect(url)
}

//renderError renders a template containing the error.
func renderError(err error, c *revel.Controller) revel.Result {

	if err != nil { //log error and send notification e-mail
		c.Log.Error("rendering error", "err", err.Error())
		app.SendErrorNote()
	}

	templatePath := "errors/render.html"

	c.ViewArgs["errMsg"] = c.Message("error.content")
	c.Validation.Keep()

	c.Log.Warn("render", "path", templatePath)
	return c.RenderTemplate(templatePath)
}

//renderQuietError renders an error message.
func renderQuietError(errType ErrorType, err error, c *revel.Controller) {

	if err != nil { //log error and send notification e-mail
		c.Log.Error("rendering quiet error", "err", err.Error())
		app.SendErrorNote()
	}

	switch errType {
	case errDB:
		c.ViewArgs["errMsg"] = c.Message("error.db")
	default:
		c.ViewArgs["errMsg"] = c.Message("error.undefined")
	}
}

//append all validation errors
func getErrorString(errs []*revel.ValidationError) (str string) {

	for i, err := range errs {
		if i != 0 {
			str += "<br>"
		}
		str += (*err).String()
	}
	return
}

//response of an ajax request
type response struct {
	Status  string
	Msg     string
	FieldID string
	Valid   bool
	Value   string
	ID      int
}

const (
	//SUCCESS result type
	SUCCESS = "success"
	//ERROR result type
	ERROR = "error"
	//INVALID validation result type
	INVALID = "invalid"
)
