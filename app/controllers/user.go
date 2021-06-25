package controllers

import (
	"database/sql"
	"strconv"
	"strings"
	"turm/app/auth"
	"turm/app/models"

	"github.com/revel/revel"
)

/*LoginPage renders the login page.
- Roles: not logged in users */
func (c User) LoginPage() revel.Result {

	c.Log.Debug("render login page", "url", c.Request.URL)

	//NOTE: we do not set the callPath because we want to be redirected to the previous page
	c.Session["currPath"] = c.Request.URL.String()
	c.Session["lastURL"] = c.Request.URL.String()

	c.ViewArgs["tab"] = c.Message("login.tab")

	return c.Render()
}

/*Login of an user.
- Roles: not logged in users */
func (c User) Login(credentials models.Credentials) revel.Result {

	c.Log.Debug("login user", "username", credentials.Username,
		"email", credentials.EMail, "stayLoggedIn", credentials.StayLoggedIn)
	c.Session["lastURL"] = c.Request.URL.String()

	credentials.Validate(c.Validation)
	if c.Validation.HasErrors() {
		return flashError(errValidation, nil, "", c.Controller, "")
	}

	var user models.User

	if credentials.Username != "" { //ldap login, authenticate the user

		success, err := auth.LDAPServerAuth(&credentials, &user)

		if err != nil {
			return flashError(errAuth, err, "", c.Controller, "")
		} else if !success {
			c.Validation.ErrorKey("login.ldap.auth.failed")
			return flashError(errValidation, nil, "", c.Controller, "")
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
		return flashError(errDB, err, "", c.Controller, "")
	}

	c.Validation.Required(user.ID).
		MessageKey("validation.invalid.login")
	if c.Validation.HasErrors() { //invalid external user credentials
		return flashError(errValidation, nil, "", c.Controller, "")
	}

	c.setSession(&user)
	c.Session["stayLoggedIn"] = strconv.FormatBool(credentials.StayLoggedIn)

	//set default expiration of session cookie
	c.Session.SetDefaultExpiration()
	if credentials.StayLoggedIn {
		c.Session.SetNoExpiration()
	}

	c.Log.Debug("login successful", "user", user)
	c.Flash.Success(c.Message("login.success",
		user.EMail,
		user.FirstName,
		user.LastName,
	))

	//not activated external users get redirected to the activation page
	if user.ActivationCode.String != "" && credentials.EMail != "" {
		c.Session["callPath"] = "/user/activationPage"
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

	c.Session["lastURL"] = c.Request.URL.String()
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
	c.Session["lastURL"] = c.Request.URL.String()

	c.ViewArgs["tab"] = c.Message("register.tab")

	return c.Render()
}

/*Registration registers a new external user and sends an activation e-mail.
- Roles: not logged in users */
func (c User) Registration(user models.User) revel.Result {

	c.Log.Debug("registration of user", "user", user)
	c.Session["lastURL"] = c.Request.URL.String()

	//register the new user
	if err := user.Register(c.Validation); err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
	} else if c.Validation.HasErrors() {
		return flashError(
			errValidation, nil, "", c.Controller, "")
	}

	c.setSession(&user)
	c.Session["notActivated"] = "true"

	data := models.EMailData{User: user}
	err := sendEMail(c.Controller, &data,
		"email.subject.activation",
		"activation")

	if err != nil {
		return flashError(errEMail, err, "/user/activationPage",
			c.Controller, user.EMail,
		)
	}

	c.Flash.Success(c.Message("register.success", user.EMail))
	return c.Redirect(User.ActivationPage)
}

/*NewPasswordPage renders the page to request a new password.
- Roles: not logged in users */
func (c User) NewPasswordPage() revel.Result {

	c.Log.Debug("render new password page", "url", c.Request.URL)

	//NOTE: we do not set the callPath because we want to be redirected to the
	//previous page after logging in
	c.Session["currPath"] = c.Request.URL.String()
	c.Session["lastURL"] = c.Request.URL.String()

	c.ViewArgs["tab"] = c.Message("new.pw.tab")

	return c.Render()
}

/*NewPassword generates a new password and sends it via e-mail.
- Roles: all */
func (c User) NewPassword(email string) revel.Result {

	c.Log.Debug("requesting new password", "email", email)
	c.Session["lastURL"] = c.Request.URL.String()

	user := models.User{EMail: strings.ToLower(email)}
	err := user.GenerateNewPassword(c.Validation)

	if err != nil {
		return flashError(errDB, err, "", c.Controller, "")
	} else if c.Validation.HasErrors() {
		return flashError(errValidation, nil, "", c.Controller, "")
	}

	mailData := models.EMailData{User: user}
	err = sendEMail(c.Controller, &mailData,
		"email.subject.new.pw",
		"newPw")

	if err != nil {
		return flashError(errEMail, err, "", c.Controller, user.EMail)
	}

	c.Flash.Success(c.Message("new.pw.success", email))
	return c.Redirect(User.LoginPage)
}

/*ActivationPage renders the activation page.
- Roles: all */
func (c User) ActivationPage() revel.Result {

	c.Log.Debug("render activation page", "url", c.Request.URL)

	//NOTE: we do not set the callPath because we want to be redirected to
	//the previous page after account activation
	c.Session["currPath"] = c.Request.URL.String()
	c.Session["lastURL"] = c.Request.URL.String()

	c.ViewArgs["tab"] = c.Message("activation.tab")

	return c.Render()
}

/*VerifyActivationCode verifies an activation code.
- Roles: all */
func (c User) VerifyActivationCode(activationCode string) revel.Result {

	c.Log.Debug("verify activation code", "activationCode", activationCode)
	c.Session["lastURL"] = c.Request.URL.String()

	//get the user ID of the to-be-activated account
	userID := c.Params.Query.Get("userID")
	if userID == "" {
		if c.Session["userID"] == nil {
			c.Validation.ErrorKey("validation.invalid.activation")
		} else {
			userID = c.Session["userID"].(string)
		}
	}

	user := models.User{ActivationCode: sql.NullString{
		String: activationCode,
		Valid:  true,
	}}
	if !c.Validation.HasErrors() {

		models.ValidateLength(&activationCode, "validation.invalid.activation",
			7, 7, c.Validation)

		user.ID, _ = strconv.Atoi(userID)
		c.Validation.Required(user.ID).
			MessageKey("validation.invalid.activation")
	}

	if c.Validation.HasErrors() {
		return flashError(errValidation, nil, "", c.Controller, "")
	}

	//set the activation code to null, if it matches
	success, err := user.VerifyActivationCode()
	if err != nil {
		return flashError(errDB, err, "", c.Controller, "")
	}

	if !success { //invalid activation code
		c.Validation.ErrorKey("validation.invalid.activation")
		return flashError(errValidation, nil, "", c.Controller, "")
	}

	c.Session.Del("notActivated")
	c.Flash.Success(c.Message("activation.success"))
	if c.Session["callPath"].(string) == "/user/activationPage" {
		return c.Redirect(App.Index)
	}
	return c.Redirect(c.Session["callPath"])
}

/*NewActivationCode sends a new activation code.
- Roles: logged in and not activated users */
func (c User) NewActivationCode() revel.Result {

	c.Log.Debug("send new activation code")
	c.Session["lastURL"] = c.Request.URL.String()

	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		return flashError(errTypeConv, err, "", c.Controller, "")
	}

	user := models.User{ID: userID}
	if err := user.NewActivationCode(); err != nil {
		return flashError(errDB, err, "", c.Controller, "")
	}

	data := models.EMailData{User: user}
	err = sendEMail(c.Controller, &data,
		"email.subject.activation",
		"activation")

	if err != nil {
		return flashError(errEMail, err, "", c.Controller, user.EMail)
	}

	c.Flash.Success(c.Message("activation.resend.success", user.EMail))
	return c.Redirect(User.ActivationPage)
}

/*PrefLanguagePage renders the page to set a preferred language.
- Roles: logged in users */
func (c User) PrefLanguagePage() revel.Result {

	c.Log.Debug("render preferred language page", "url", c.Request.URL)

	//NOTE: we do not set the callPath because we want to be redirected to the previous
	//page after a successful login
	c.Session["currPath"] = c.Request.URL.String()
	c.Session["lastURL"] = c.Request.URL.String()

	c.ViewArgs["tab"] = c.Message("pref.lang.tab")

	return c.Render()
}

/*SetPrefLanguage sets the preferred language of the user.
- Roles: logged in users */
func (c User) SetPrefLanguage(prefLanguage string) revel.Result {

	c.Log.Debug("set preferred language", "prefLanguage", prefLanguage)
	c.Session["lastURL"] = c.Request.URL.String()

	c.Validation.Check(prefLanguage,
		models.LanguageValidator{},
	)

	if c.Validation.HasErrors() {
		return flashError(errValidation, nil, "", c.Controller, "")
	}

	//get the user ID
	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		return flashError(errTypeConv, err, "", c.Controller, "")
	}

	user := models.User{ID: userID,
		Language: sql.NullString{
			String: prefLanguage,
			Valid:  true,
		}}

	//update the language
	if err := user.SetPrefLanguage(); err != nil {
		return flashError(errDB, err, "", c.Controller, "")
	}

	c.Flash.Success(c.Message("pref.lang.success", user.Language.String))
	return c.Redirect(c.Session["callPath"])
}

/*Profile page of the user.
- Roles: logged in and activated users */
func (c User) Profile() revel.Result {

	c.Log.Debug("render profile page")

	c.Session["callPath"] = c.Request.URL.String()
	c.Session["currPath"] = c.Request.URL.String()
	c.Session["lastURL"] = c.Request.URL.String()

	c.ViewArgs["tab"] = c.Message("profile.tab")

	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		renderQuietError(errTypeConv, err, c.Controller)
		return c.Render()
	}

	user := models.User{ID: userID}
	if err = user.GetProfileData(); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	return c.Render(user)
}

/*ChangePassword of an user.
- Roles: logged in and activated users */
func (c User) ChangePassword(oldPw, newPw1, newPw2 string) revel.Result {

	c.Log.Debug("change password of user", "oldPw", oldPw, "newPw1", newPw1,
		"newPw2", newPw2)
	c.Session["lastURL"] = c.Request.URL.String()

	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		renderQuietError(errTypeConv, err, c.Controller)
		return c.Render()
	}

	user := models.User{ID: userID,
		Password: sql.NullString{Valid: true, String: oldPw}}
	err = user.NewPassword(newPw1, newPw2, c.Validation)

	if err != nil {
		return flashError(errDB, err, "", c.Controller, "")
	} else if c.Validation.HasErrors() {
		return flashError(errValidation, nil, "", c.Controller, "")
	}

	mailData := models.EMailData{User: user}
	err = sendEMail(c.Controller, &mailData,
		"email.subject.change.pw",
		"changePw")

	if err != nil {
		return flashError(errEMail, err, "", c.Controller, user.EMail)
	}

	c.Flash.Success(c.Message("profile.change.pw.success"))
	return c.Redirect(User.Profile)
}

//setSession sets all user related session values.
func (c User) setSession(user *models.User) {

	c.Log.Debug("setting user session", "user", user)
	c.Session["userID"] = strconv.Itoa(user.ID)
	c.Session["firstName"] = user.FirstName
	c.Session["lastName"] = user.LastName
	c.Session["role"] = user.Role.String()
	c.Session["isEditor"] = strconv.FormatBool(user.IsEditor)
	c.Session["isInstructor"] = strconv.FormatBool(user.IsInstructor)
	c.Session["eMail"] = user.EMail
	c.Session["prefLanguage"] = user.Language.String
}
