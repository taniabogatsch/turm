package controllers

import (
	"turm/app/models"

	"github.com/revel/revel"
)

/*Index renders the landing page of the application.
- Roles: all (except not activated users) */
func (c App) Index() revel.Result {

	c.Log.Debug("render app index page", "url", c.Request.URL)

	c.Session["callPath"] = c.Request.URL.String()
	c.Session["currPath"] = c.Request.URL.String()
	c.Session["lastURL"] = c.Request.URL.String()

	c.ViewArgs["tab"] = c.Message("index.tab")

	return c.Render()
}

/*Groups renders all groups and their children recursively.
- Roles: all (except not activated users) */
func (c App) Groups(prefix string) revel.Result {

	c.Log.Debug("render groups", "prefix", prefix)
	c.Session["lastURL"] = c.Request.URL.String()

	//NOTE: no prefix validation, if this controller is called with an
	//invalid prefix, then something else is going wrong

	//TODO: why is Groups in upper case?

	var Groups models.Groups
	if err := Groups.Select(&prefix); err != nil {
		return renderError(err, c.Controller)
	}

	return c.Render(Groups)
}

/*ChangeLanguage changes the language, then redirects to the page currently set as currPath.
- Roles: all */
func (c App) ChangeLanguage(language string) revel.Result {

	c.Log.Debug("change language", "old language", c.Session["currentLocale"],
		"language", language)
	c.Session["lastURL"] = c.Request.URL.String()

	c.Validation.Check(language,
		models.LanguageValidator{},
	).MessageKey("validation.invalid.language")

	if c.Validation.HasErrors() {
		return flashError(
			errValidation, nil, "", c.Controller, "")
	}

	c.Session["currentLocale"] = language
	c.ViewArgs["currentLocale"] = c.Session["currentLocale"]
	c.Request.Locale = c.Session["currentLocale"].(string)

	c.Flash.Success(c.Message("language.change.success",
		language,
	))
	return c.Redirect(c.Session["currPath"])
}

/*DataPrivacy renders the data privacy page.
- Roles: all (except not activated users) */
func (c App) DataPrivacy() revel.Result {

	c.Log.Debug("render data privacy page", "url", c.Request.URL)

	c.Session["callPath"] = c.Request.URL.String()
	c.Session["currPath"] = c.Request.URL.String()
	c.Session["lastURL"] = c.Request.URL.String()

	c.ViewArgs["tab"] = c.Message("data.privacy.tab")

	return c.Render()
}

/*Imprint renders the imprint page.
- Roles: all (except not activated users) */
func (c App) Imprint() revel.Result {

	c.Log.Debug("render imprint page", "url", c.Request.URL)

	c.Session["callPath"] = c.Request.URL.String()
	c.Session["currPath"] = c.Request.URL.String()
	c.Session["lastURL"] = c.Request.URL.String()

	c.ViewArgs["tab"] = c.Message("imprint.tab")

	return c.Render()
}

/*News renders the news feed page.
- Roles: all (except not activated users) */
func (c App) News() revel.Result {

	c.Log.Debug("render news feed page", "url", c.Request.URL)

	c.Session["callPath"] = c.Request.URL.String()
	c.Session["currPath"] = c.Request.URL.String()
	c.Session["lastURL"] = c.Request.URL.String()

	c.ViewArgs["tab"] = c.Message("news.feed.tab")

	var categories models.Categories
	if err := categories.Select(tabNewsFeedCategory); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	return c.Render(categories)
}

/*FAQs renders the FAQs page.
- Roles: all (except not activated users) */
func (c App) FAQs() revel.Result {

	c.Log.Debug("render FAQs page", "url", c.Request.URL)

	c.Session["callPath"] = c.Request.URL.String()
	c.Session["currPath"] = c.Request.URL.String()
	c.Session["lastURL"] = c.Request.URL.String()

	c.ViewArgs["tab"] = c.Message("faq.tab")

	var categories models.Categories
	if err := categories.Select(tabFAQCategory); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	return c.Render(categories)
}
