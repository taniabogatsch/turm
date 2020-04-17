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
		revel.AppLog.Debug("ldap authentication successful", "user", user)

	} else { //external login

		user.EMail = strings.ToLower(credentials.EMail)
		user.Password.String = credentials.Password
		user.Password.Valid = true
		revel.AppLog.Debug("login of extern user", "user", user)
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

	c.setSession(&user)
	c.Session["stayLoggedIn"] = strconv.FormatBool(credentials.StayLoggedIn)

	//set default expiration of session cookie
	c.Session.SetDefaultExpiration()
	if credentials.StayLoggedIn {
		c.Session.SetNoExpiration()
	}

	revel.AppLog.Debug("login successful", "user", user)
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

	err := c.sendEMail(&user,
		"emails.subject_activation",
		"activation")
	if err != nil {
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

/*NewPassword generates a new password and sends it via e-mail.
- Roles: all */
func (c User) NewPassword(email string) revel.Result {

	revel.AppLog.Debug("requesting new password", "email", email)

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
			routes.User.NewPasswordPage(),
			"",
			c.Controller,
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
		c.Flash.Success(c.Message("newPw.confirmation_success", email))
		return c.Redirect(User.LoginPage)
	}

	user := models.User{EMail: strings.ToLower(email)}
	if err := db.NewPassword(&user); err != nil {
		return flashError(
			errDB,
			routes.User.NewPasswordPage(),
			"",
			c.Controller,
		)
	}
	revel.AppLog.Debug("set new password", "user", user)

	err := c.sendEMail(&user,
		"emails.subject_newPw",
		"newPw")
	if err != nil {
		return flashError(
			errEMail,
			routes.User.NewPasswordPage(),
			"",
			c.Controller,
		)
	}

	c.Flash.Success(c.Message("newPw.confirmation_success", email))
	return c.Redirect(User.LoginPage)
}

/*ActivationPage renders the activation page.
- Roles: all */
func (c User) ActivationPage() revel.Result {

	revel.AppLog.Debug("requesting activation page")
	//NOTE: we do not set the callPath because we want to be redirected to
	//the previous page after account activation
	c.Session["currPath"] = c.Request.URL.String()
	c.ViewArgs["tabName"] = c.Message("activation.tabName")
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

	var uID int
	if !c.Validation.HasErrors() {
		c.Validation.Check(activationCode,
			revel.Required{},
			revel.MinSize{7},
			revel.MaxSize{7},
		).MessageKey("validation.invalid.activation")

		uID, _ = strconv.Atoi(userID)
		c.Validation.Required(uID).
			MessageKey("validation.invalid.activation")
	}

	if c.Validation.HasErrors() {
		return flashError(
			errValidation,
			routes.User.ActivationPage(),
			"",
			c.Controller,
		)
	}

	//set the activation code to null, if it matches
	success, err := db.VerifyActivationCode(&activationCode, &uID)
	if err != nil {
		return flashError(
			errDB,
			routes.User.ActivationPage(),
			"",
			c.Controller,
		)
	}

	if !success { //invalid activation code
		c.Validation.ErrorKey("validation.invalid.activation")
		return flashError(
			errValidation,
			routes.User.ActivationPage(),
			"",
			c.Controller,
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
			errDataConversion,
			routes.User.ActivationPage(),
			"",
			c.Controller,
		)
	}

	if err := db.NewActivationCode(&user); err != nil {
		return flashError(
			errDB,
			routes.User.ActivationPage(),
			"",
			c.Controller,
		)
	}

	err = c.sendEMail(&user,
		"emails.subject_activation",
		"activation")
	if err != nil {
		return flashError(
			errEMail,
			routes.User.ActivationPage(),
			"",
			c.Controller,
		)
	}

	c.Flash.Success(c.Message("activation.resendCode_success"))
	return c.Redirect(User.ActivationPage)
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
	if err := db.SetPrefLanguage(&userID, &prefLanguage); err != nil {
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

//sendEMail sends an activation e-mail or a new password e-mail.
func (c User) sendEMail(user *models.User, subjectKey string, filename string) (err error) {

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
		subjectKey,
		filename,
		&email,
		c.Controller,
	)
	if err != nil {
		return
	}

	revel.AppLog.Debug("assembled e-mail", "email", email)

	app.EMailQueue <- email
	return
}
