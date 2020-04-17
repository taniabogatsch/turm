package controllers

import (
	"strconv"
	"strings"
	"turm/app"
	"turm/app/auth"
	"turm/app/db"
	"turm/app/models"
	"turm/app/routes"

	"github.com/revel/revel"
)

/*LoginPage renders the login page.
- Roles: not logged in users */
func (c User) LoginPage() revel.Result {

	revel.AppLog.Debug("requesting login page")
	//NOTE: we do not set the callPath because we want to be redirected to the previous page
	c.Session["currPath"] = c.Request.URL.String()
	c.ViewArgs["tabName"] = c.Message("login.tabName")
	return c.Render()
}

/*Login implements the login of an user. It redirects to callPath.
- Roles: not logged in users */
func (c User) Login(credentials models.Credentials) revel.Result {

	revel.AppLog.Debug("login user", "username", credentials.Username, "email", credentials.EMail,
		"stayLoggedIn", credentials.StayLoggedIn)

	credentials.Validate(c.Validation)
	if c.Validation.HasErrors() {
		return flashError(
			errValidation,
			routes.User.LoginPage(),
			"",
			c.Controller,
		)
	}

	var user models.User

	if credentials.Username != "" { //ldap login, authenticate the user

		err := auth.LDAPServerAuth(&credentials, &user)
		if err != nil {
			c.Validation.ErrorKey("login.ldapAuthentication_invalid_danger")
			return flashError(
				errValidation,
				routes.User.LoginPage(),
				"",
				c.Controller,
			)
		}
		revel.AppLog.Debug("authentication successful", "user", user)

	} else { //external login

		user.EMail = strings.ToLower(credentials.EMail)
		user.Password.String = credentials.Password
		user.Password.Valid = true
	}

	//login of user
	if err := db.Login(&user); err != nil {
		return flashError(
			errDB,
			routes.User.LoginPage(),
			"",
			c.Controller,
		)
	}

	c.Validation.Required(user.ID).
		MessageKey("validation.invalid.login")
	if c.Validation.HasErrors() { //invalid external user credentials
		return flashError(
			errValidation,
			routes.User.LoginPage(),
			"",
			c.Controller,
		)
	}

	revel.AppLog.Debug("login successful", "user", user)
	c.setSession(&user)
	c.Session["stayLoggedIn"] = strconv.FormatBool(credentials.StayLoggedIn)

	//set default expiration of session cookie
	c.Session.SetDefaultExpiration()
	if credentials.StayLoggedIn {
		c.Session.SetNoExpiration()
	}

	c.Flash.Success(c.Message("login.confirmation_success"))

	//not activated external users get redirected to the activation page
	if user.ActivationCode.String != "" && credentials.EMail != "" {
		c.Session["callPath"] = routes.User.ActivationPage()
		c.Session["notActivated"] = "true"
	}

	//if not yet set, prompt the user to set the preferred language
	if !user.Language.Valid {
		return c.Redirect(User.PrefLanguagePage)
	}

	return c.Redirect(c.Session["callPath"])
}

/*Logout handles logout, deletes all session values.
- Roles: all */
func (c User) Logout() revel.Result {

	revel.AppLog.Debug("logout", "length session", len(c.Session))
	for k := range c.Session {
		c.Session.Del(k)
	}

	c.Flash.Success(c.Message("logout.success"))
	return c.Redirect(User.LoginPage)
}

/*RegistrationPage renders the registration page.
- Roles: not logged in users */
func (c User) RegistrationPage() revel.Result {

	revel.AppLog.Debug("requesting registration page")
	//NOTE: we do not set the callPath because we want to be redirected to
	//the previous page after account activation
	c.ViewArgs["tabName"] = c.Message("register.tabName")
	c.Session["currPath"] = c.Request.URL.String()
	return c.Render()
}

/*Registration registers a new external user and sends an activation e-mail.
- Roles: not logged in users */
func (c User) Registration(user models.User) revel.Result {

	revel.AppLog.Debug("registration of user", "user", user)

	user.Validate(c.Validation)
	if c.Validation.HasErrors() {
		return flashError(
			errValidation,
			routes.User.RegistrationPage(),
			"",
			c.Controller,
		)
	}

	//register the new user
	if err := db.Register(&user); err != nil {
		return flashError(
			errDB,
			routes.User.RegistrationPage(),
			"",
			c.Controller,
		)
	}
	revel.AppLog.Debug("registration successful", "user", user)

	c.setSession(&user)
	c.Session["notActivated"] = "true"

	if err := c.sendActivationEMail(&user); err != nil {
		return flashError(
			errEMail,
			routes.User.ActivationPage(),
			"",
			c.Controller,
		)
	}

	c.Flash.Success(c.Message("activation.codeSend_info"))
	return c.Redirect(User.ActivationPage)
}

/*NewPasswordPage renders the page to request a new password.
- Roles: not logged in users */
func (c User) NewPasswordPage() revel.Result {

	revel.AppLog.Debug("requesting new password page")
	//NOTE: we do not set the callPath because we want to be redirected to the
	//previous page after logging in
	c.Session["currPath"] = c.Request.URL.String()
	c.ViewArgs["tabName"] = c.Message("newPw.tabName")
	return c.Render()
}

/*ActivationPage renders the activation page.
- Roles: logged in and not activated users */
func (c User) ActivationPage() revel.Result {

	revel.AppLog.Debug("requesting activation page")
	//NOTE: we do not set the callPath because we want to be redirected to
	//the previous page after account activation
	c.Session["currPath"] = c.Request.URL.String()
	c.ViewArgs["tabName"] = c.Message("activation.tabName")
	return c.Render()
}

/*PrefLanguagePage renders the page to set a preferred language.
- Roles: logged in users. */
func (c User) PrefLanguagePage() revel.Result {

	revel.AppLog.Debug("requesting preferred language page")
	//NOTE: we do not set the callPath because we want to be redirected to the previous
	//page after a successful login
	c.Session["currPath"] = c.Request.URL.String()
	c.ViewArgs["tabName"] = c.Message("prefLang.tabName")
	return c.Render()
}

/*SetPrefLanguage sets the preferred language of the user.
- Roles: logged in users. */
func (c User) SetPrefLanguage(prefLanguage string) revel.Result {

	revel.AppLog.Debug("set preferred language", "prefLanguage", prefLanguage)

	c.Validation.Check(prefLanguage,
		models.LanguageValidator{},
	)
	if c.Validation.HasErrors() {
		return flashError(
			errValidation,
			routes.User.PrefLanguagePage(),
			"",
			c.Controller,
		)
	}

	//update the language
	userID := c.Session["userID"].(string)
	if err := db.SetPrefLanguage(userID, prefLanguage); err != nil {
		return flashError(
			errDB,
			routes.User.PrefLanguagePage(),
			"",
			c.Controller,
		)
	}
	return c.Redirect(c.Session["callPath"])
}

//setSession sets all user related session values.
func (c User) setSession(user *models.User) {

	c.Session["userID"] = strconv.Itoa(user.ID)
	c.Session["firstName"] = user.FirstName
	c.Session["lastName"] = user.LastName
	c.Session["role"] = user.Role.String()
	c.Session["eMail"] = user.EMail
	c.Session["prefLanguage"] = user.Language.String
}

//sendActivationEMail sends an e-mail with an activation code and an activation URL. */
func (c User) sendActivationEMail(user *models.User) (err error) {

	data := models.EMailData{User: *user}

	if !user.Language.Valid {
		user.Language.String = app.DefaultLanguage
	}

	email := app.EMail{
		Recipient: user.EMail,
		ReplyTo:   c.Message("mails.doNotReply", app.ServiceEMail),
	}

	err = models.GetEMailSubjectBody(
		&data,
		&user.Language.String,
		"emails.subject_activation",
		"activation",
		&email,
		c.Controller,
	)
	if err != nil {
		return
	}

	revel.AppLog.Debug("assembled e-mail", "e-mail", email)

	app.EMailQueue <- email
	return
}
