package controllers

import (
	"database/sql"
	"strconv"
	"strings"
	"turm/app"
	"turm/app/auth"
	"turm/app/models"
	"turm/app/routes"

	"github.com/revel/revel"
)

/*LoginPage renders the login page.
- Roles: not logged in users */
func (c User) LoginPage() revel.Result {

	c.Log.Debug("render login page", "url", c.Request.URL)
	//NOTE: we do not set the callPath because we want to be redirected to the previous page
	c.Session["currPath"] = c.Request.URL.String()
	c.ViewArgs["tabName"] = c.Message("login.tab")
	return c.Render()
}

/*Login implements the login of an user. It redirects to callPath.
- Roles: not logged in users */
func (c User) Login(credentials models.Credentials) revel.Result {

	c.Log.Debug("login user", "username", credentials.Username,
		"email", credentials.EMail, "stayLoggedIn", credentials.StayLoggedIn)

	credentials.Validate(c.Validation)
	if c.Validation.HasErrors() {
		return flashError(
			errValidation,
			nil,
			routes.User.LoginPage(),
			c.Controller,
			"",
		)
	}

	var user models.User

	if credentials.Username != "" { //ldap login, authenticate the user

		err := auth.LDAPServerAuth(&credentials, &user)
		if err != nil {
			c.Validation.ErrorKey("login.ldap.auth.failed")
			return flashError(
				errAuth,
				err,
				routes.User.LoginPage(),
				c.Controller,
				"",
			)
		}
		c.Log.Debug("ldap authentication successful", "user", user)

	} else { //external login

		user.EMail = strings.ToLower(credentials.EMail)
		user.Password.String = credentials.Password
		user.Password.Valid = true
		c.Log.Debug("login of external user", "user", user)
	}

	//login of user
	if err := user.Login(); err != nil {
		return flashError(
			errDB,
			err,
			routes.User.LoginPage(),
			c.Controller,
			"",
		)
	}

	c.Validation.Required(user.ID).
		MessageKey("validation.invalid.login")
	if c.Validation.HasErrors() { //invalid external user credentials
		return flashError(
			errValidation,
			nil,
			routes.User.LoginPage(),
			c.Controller,
			"",
		)
	}

	c.setSession(&user)
	c.Session["stayLoggedIn"] = strconv.FormatBool(credentials.StayLoggedIn)

	//set default expiration of session cookie
	c.Session.SetDefaultExpiration()
	if credentials.StayLoggedIn {
		c.Session.SetNoExpiration()
	}

	c.Log.Debug("login successful", "user", user)
	c.Flash.Success(
		c.Message(
			"login.success",
			user.EMail,
			user.FirstName,
			user.LastName,
		))

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

	c.Log.Debug("logout", "length session", len(c.Session))
	for k := range c.Session {
		c.Session.Del(k)
	}
	c.Flash.Success(c.Message("logout.success"))
	return c.Redirect(User.LoginPage)
}

/*RegistrationPage renders the registration page.
- Roles: not logged in users */
func (c User) RegistrationPage() revel.Result {

	c.Log.Debug("render registration page", "url", c.Request.URL)
	//NOTE: we do not set the callPath because we want to be redirected to
	//the previous page after account activation
	c.Session["currPath"] = c.Request.URL.String()
	c.ViewArgs["tabName"] = c.Message("register.tab")
	return c.Render()
}

/*Registration registers a new external user and sends an activation e-mail.
- Roles: not logged in users */
func (c User) Registration(user models.User) revel.Result {

	c.Log.Debug("registration of user", "user", user)

	user.Validate(c.Validation)
	if c.Validation.HasErrors() {
		return flashError(
			errValidation,
			nil,
			routes.User.RegistrationPage(),
			c.Controller,
			"",
		)
	}

	//register the new user
	if err := user.Register(); err != nil {
		return flashError(
			errDB,
			err,
			routes.User.RegistrationPage(),
			c.Controller,
			"",
		)
	}
	c.Log.Debug("registration successful", "user", user)

	c.setSession(&user)
	c.Session["notActivated"] = "true"

	err := c.sendEMail(&user,
		"email.subject.activation",
		"activation")
	if err != nil {
		return flashError(
			errEMail,
			err,
			routes.User.ActivationPage(),
			c.Controller,
			user.EMail,
		)
	}

	c.Flash.Success(
		c.Message(
			"register.success",
			user.EMail,
		))
	return c.Redirect(User.ActivationPage)
}

/*NewPasswordPage renders the page to request a new password.
- Roles: not logged in users */
func (c User) NewPasswordPage() revel.Result {

	c.Log.Debug("render new password page", "url", c.Request.URL)
	//NOTE: we do not set the callPath because we want to be redirected to the
	//previous page after logging in
	c.Session["currPath"] = c.Request.URL.String()
	c.ViewArgs["tabName"] = c.Message("new.pw.tab")
	return c.Render()
}

/*NewPassword generates a new password and sends it via e-mail.
- Roles: all */
func (c User) NewPassword(email string) revel.Result {

	c.Log.Debug("requesting new password", "email", email)

	c.Validation.Check(email,
		revel.Required{},
		revel.MaxSize{255},
	).MessageKey("validation.invalid.email")
	c.Validation.Email(email).
		MessageKey("validation.invalid.email")

	isLdapEMail := !strings.Contains(strings.ToLower(email), app.EMailSuffix)
	c.Validation.Required(isLdapEMail).
		MessageKey("validation.email.ldap")

	if c.Validation.HasErrors() {
		return flashError(
			errValidation,
			nil,
			routes.User.NewPasswordPage(),
			c.Controller,
			"",
		)
	}

	//we do not want to provide any information on whether an e-mail exists
	data := models.ValidateUniqueData{
		Column: "email",
		Table:  "users",
		Value:  strings.ToLower(email),
	}
	c.Validation.Check(data,
		models.NotUnique{},
	)
	if c.Validation.HasErrors() {
		c.Flash.Success(
			c.Message(
				"new.pw.success",
				email,
			))
		return c.Redirect(User.LoginPage)
	}

	user := models.User{EMail: strings.ToLower(email)}
	if err := user.NewPassword(); err != nil {
		return flashError(
			errDB,
			err,
			routes.User.NewPasswordPage(),
			c.Controller,
			"",
		)
	}
	c.Log.Debug("set new password", "user", user)

	err := c.sendEMail(&user,
		"email.subject.new.pw",
		"newPw")
	if err != nil {
		return flashError(
			errEMail,
			err,
			routes.User.NewPasswordPage(),
			c.Controller,
			user.EMail,
		)
	}

	c.Flash.Success(
		c.Message(
			"new.pw.success",
			email,
		))
	return c.Redirect(User.LoginPage)
}

/*ActivationPage renders the activation page.
- Roles: all */
func (c User) ActivationPage() revel.Result {

	c.Log.Debug("render activation page", "url", c.Request.URL)
	//NOTE: we do not set the callPath because we want to be redirected to
	//the previous page after account activation
	c.Session["currPath"] = c.Request.URL.String()
	c.ViewArgs["tabName"] = c.Message("activation.tab")
	return c.Render()
}

/*VerifyActivationCode verifies an activation code.
- Roles: all */
func (c User) VerifyActivationCode(activationCode string) revel.Result {

	//get the user ID of the to-be-activated account
	userID := c.Params.Query.Get("userID")
	if userID == "" {
		if c.Session["userID"] == nil {
			c.Validation.ErrorKey("validation.invalid.activation")
		} else {
			userID = c.Session["userID"].(string)
		}
	}

	user := models.User{ActivationCode: sql.NullString{activationCode, true}}
	if !c.Validation.HasErrors() {
		c.Validation.Check(activationCode,
			revel.MinSize{7},
			revel.MaxSize{7},
		).MessageKey("validation.invalid.activation")

		user.ID, _ = strconv.Atoi(userID)
		c.Validation.Required(user.ID).
			MessageKey("validation.invalid.activation")
	}

	if c.Validation.HasErrors() {
		return flashError(
			errValidation,
			nil,
			routes.User.ActivationPage(),
			c.Controller,
			"",
		)
	}

	//set the activation code to null, if it matches
	success, err := user.VerifyActivationCode()
	if err != nil {
		return flashError(
			errDB,
			err,
			routes.User.ActivationPage(),
			c.Controller,
			"",
		)
	}

	if !success { //invalid activation code
		c.Validation.ErrorKey("validation.invalid.activation")
		return flashError(
			errValidation,
			nil,
			routes.User.ActivationPage(),
			c.Controller,
			"",
		)
	}

	c.Session.Del("notActivated")
	c.Flash.Success(c.Message("activation.success"))
	if c.Session["callPath"].(string) == routes.User.ActivationPage() {
		return c.Redirect(App.Index)
	}
	return c.Redirect(c.Session["callPath"])
}

/*NewActivationCode sends a new activation code.
- Roles: logged in and not activated users */
func (c User) NewActivationCode() revel.Result {

	userID := c.Session["userID"].(string)
	var user models.User
	var err error

	if user.ID, err = strconv.Atoi(userID); err != nil {
		return flashError(
			errTypeConv,
			err,
			routes.User.ActivationPage(),
			c.Controller,
			"",
		)
	}

	if err := user.NewActivationCode(); err != nil {
		return flashError(
			errDB,
			err,
			routes.User.ActivationPage(),
			c.Controller,
			"",
		)
	}

	err = c.sendEMail(&user,
		"email.subject.activation",
		"activation")
	if err != nil {
		return flashError(
			errEMail,
			err,
			routes.User.ActivationPage(),
			c.Controller,
			user.EMail,
		)
	}

	c.Flash.Success(
		c.Message(
			"activation.resend.success",
			user.EMail,
		))
	return c.Redirect(User.ActivationPage)
}

/*PrefLanguagePage renders the page to set a preferred language.
- Roles: logged in users. */
func (c User) PrefLanguagePage() revel.Result {

	c.Log.Debug("render preferred language page", "url", c.Request.URL)
	//NOTE: we do not set the callPath because we want to be redirected to the previous
	//page after a successful login
	c.Session["currPath"] = c.Request.URL.String()
	c.ViewArgs["tabName"] = c.Message("pref.lang.tab")
	return c.Render()
}

/*SetPrefLanguage sets the preferred language of the user.
- Roles: logged in users. */
func (c User) SetPrefLanguage(prefLanguage string) revel.Result {

	c.Log.Debug("set preferred language", "prefLanguage", prefLanguage)

	c.Validation.Check(prefLanguage,
		models.LanguageValidator{},
	)
	if c.Validation.HasErrors() {
		return flashError(
			errValidation,
			nil,
			routes.User.PrefLanguagePage(),
			c.Controller,
			"",
		)
	}

	//update the language
	userID := c.Session["userID"].(string)
	user := models.User{Language: sql.NullString{prefLanguage, true}}
	if err := user.SetPrefLanguage(&userID); err != nil {
		return flashError(
			errDB,
			err,
			routes.User.PrefLanguagePage(),
			c.Controller,
			"",
		)
	}

	c.Flash.Success(c.Message("pref.lang.success", prefLanguage))
	return c.Redirect(c.Session["callPath"])
}

//setSession sets all user related session values.
func (c User) setSession(user *models.User) {

	c.Log.Debug("setting user session", "user", user)
	c.Session["userID"] = strconv.Itoa(user.ID)
	c.Session["firstName"] = user.FirstName
	c.Session["lastName"] = user.LastName
	c.Session["role"] = user.Role.String()
	c.Session["eMail"] = user.EMail
	c.Session["prefLanguage"] = user.Language.String
}

//sendEMail sends an activation e-mail or a new password e-mail.
func (c User) sendEMail(user *models.User, subjectKey string, filename string) (err error) {

	c.Log.Debug("sending EMail", "user", user, "subjectKey", subjectKey,
		"filename", filename)

	data := models.EMailData{User: *user}

	if !user.Language.Valid {
		user.Language.String = app.DefaultLanguage
	}

	email := app.EMail{
		Recipient: user.EMail,
	}

	err = models.GetEMailSubjectBody(
		&data,
		&user.Language.String,
		subjectKey,
		filename,
		&email,
		c.Controller,
	)
	if err != nil {
		return
	}

	c.Log.Debug("assembled e-mail", "email", email)

	app.EMailQueue <- email
	return
}
