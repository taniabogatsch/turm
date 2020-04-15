package controllers

import (
	"strconv"
	"strings"
	"turm/app"
	"turm/app/auth"
	"turm/app/database"
	"turm/app/models"

	"github.com/revel/revel"
)

/*LoginPage renders the login page.
- Roles: not logged in users */
func (c User) LoginPage() revel.Result {

	revel.AppLog.Debug("requesting login page")
	//NOTE: we do not set the callPath because we want to be redirected to e.g. a course after a login
	c.Session["currPath"] = c.Message("login.tabName")
	c.ViewArgs["tabName"] = c.Message("login.tabName")
	return c.Render()
}

/*Login implements the login of an user. It redirects to callPath.
- Roles: not logged in users */
func (c User) Login(credentials models.Credentials) revel.Result {

	revel.AppLog.Debug("login user", "username", credentials.Username, "email", credentials.EMail,
		"stayLoggedIn", credentials.StayLoggedIn)
	if credentials.ValidateCredentials(c.Validation); c.Validation.HasErrors() {
		return flashError(errValidation, c.Controller, "/User/LoginPage", "")
	}

	var user models.User

	if credentials.Username != "" { //ldap login, authenticate the user

		if err := auth.LDAPServerAuth(&credentials, &user); err != nil {
			return flashError(errAuth, c.Controller, "/User/LoginPage", "login.ldapAuthentication_invalid_danger")
		}
		revel.AppLog.Debug("authentication successful", "user", user)

	} else { //external login

		user.EMail = strings.ToLower(credentials.EMail)
		user.Password.String = credentials.Password
		user.Password.Valid = true
	}

	//login of user
	if err := database.Login(&user); err != nil {
		return flashError(errDB, c.Controller, "/User/LoginPage", "")
	}
	revel.AppLog.Debug("login successful", "user", user)

	setSession(&user, c.Controller)
	c.Session["stayLoggedIn"] = strconv.FormatBool(credentials.StayLoggedIn)

	//set default expiration of session cookie
	if !credentials.StayLoggedIn {
		c.Session.SetDefaultExpiration()
	} else {
		c.Session.SetNoExpiration()
	}

	c.Flash.Success(c.Message("login.confirmation_success"))

	//not activated external users get redirected to the activation page
	if user.ActivationCode.String != "" && credentials.EMail != "" {
		c.Session["notActivated"] = "true"
		return c.Redirect(User.ActivationPage, user.ID)
	}
	return c.Redirect(App.Index)
}

/*Logout handles logout, deletes all session values.
- Roles: all */
func (c User) Logout() revel.Result {

	revel.AppLog.Debug("logout", "length session", len(c.Session))
	for k := range c.Session {
		if k != "currentLocale" {
			c.Session.Del(k)
		}
	}

	revel.AppLog.Debug("logout successful", "length session", len(c.Session))
	c.Flash.Success(c.Message("logout.success"))
	return c.Redirect(User.LoginPage)
}

/*RegistrationPage renders the registration page.
- Roles: not logged in users */
func (c User) RegistrationPage() revel.Result {

	revel.AppLog.Debug("requesting registration page")
	//NOTE: we do not set the callPath because we want to be redirected to the activation page
	c.Session["currPath"] = c.Message("register.tabName")
	c.ViewArgs["tabName"] = c.Message("register.tabName")
	return c.Render()
}

/*Registration registers a new external user and sends an activation e-mail.
- Roles: not logged in users */
func (c User) Registration(user models.User) revel.Result {

	revel.AppLog.Debug("registration of user", "user", user)
	if user.ValidateUser(c.Validation); c.Validation.HasErrors() {
		return flashError(errValidation, c.Controller, "/User/RegistrationPage", "")
	}

	//register the new user
	if err := database.Register(&user); err != nil {
		return flashError(errDB, c.Controller, "/User/RegistrationPage", "")
	}
	revel.AppLog.Debug("registration successful", "user", user)

	//TODO: send the activation e-mail

	setSession(&user, c.Controller)
	c.Session["notActivated"] = "true"

	c.Flash.Success(c.Message("activation.codeSend_info"))
	return c.Redirect(User.ActivationPage, user.ID)
}

/*NewPasswordPage renders the page to request a new password.
- Roles: not logged in users */
func (c User) NewPasswordPage() revel.Result {

	revel.AppLog.Debug("requesting new password page")
	//NOTE: we do not set the callPath because we want to be redirected to the login page
	c.Session["currPath"] = c.Message("newPw.tabName")
	c.ViewArgs["tabName"] = c.Message("newPw.tabName")
	return c.Render()
}

/*ActivationPage renders the activation page.
- Roles: logged in and not activated users */
func (c User) ActivationPage(userID int) revel.Result {

	revel.AppLog.Debug("requesting activation page")
	c.Session["currPath"] = c.Message("activation.tabName")
	c.Session["callPath"] = "/User/ActivationPage?userID=" + strconv.Itoa(userID)
	c.ViewArgs["tabName"] = c.Message("activation.tabName")
	return c.Render(userID)
}

//setSession sets all user related session values.
func setSession(user *models.User, c *revel.Controller) {

	c.Session["userID"] = strconv.Itoa(user.ID)
	c.Session["firstName"] = user.FirstName
	c.Session["lastName"] = user.LastName
	c.Session["role"] = user.Role.String()
	c.Session["eMail"] = user.EMail
}

//sendActivationEMail sends an e-mail with an activation code and an activation URL. */
func sendActivationEMail(c *revel.Controller, subjectKey string, user *models.User) (err error) {

	data := models.EMailData{User: *user}

	//TODO: get the subject in the default language of the user
	subject := ""

	//TODO: set e-mail path
	templatePath := ""

	email := app.EMail{
		Recipient: user.EMail,
		Subject:   subject,
		ReplyTo:   c.Message("mails.doNotReply", app.ServiceEMail),
	}

	if err = models.GetEMailBody(&data, templatePath, &email.Body, c); err != nil {
		return
	}

	app.AddEMailToQueue(&email)
	return
}
