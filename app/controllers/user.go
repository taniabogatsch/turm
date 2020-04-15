package controllers

import (
	"strconv"
	"strings"
	"turm/app/auth"
	"turm/app/database"
	"turm/app/models"

	"github.com/revel/revel"
)

/*User implements logic to CRUD users. */
type User struct {
	*revel.Controller
}

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
		c.Validation.Keep()
		return c.Redirect(User.LoginPage)
	}

	var user models.User

	if credentials.Username != "" {
		//ldap login, authenticate the user
		if err := auth.LDAPServerAuth(&credentials, &user); err != nil {
			c.Flash.Error(c.Message("login.ldapAuthentication_invalid_danger"))
			return c.Redirect(User.LoginPage)
		}
		revel.AppLog.Debug("authentication successful", "user", user)
	} else {
		//external login
		user.EMail = strings.ToLower(credentials.EMail)
		user.Password.String = credentials.Password
		user.Password.Valid = true
	}

	//login of user
	if err := database.Login(&user); err != nil {
		c.Flash.Error(c.Message("error.database"))
		return c.Redirect(User.LoginPage)
	}
	revel.AppLog.Debug("login successful", "user", user)

	c.Session["userID"] = strconv.Itoa(user.ID)
	c.Session["firstName"] = user.FirstName
	c.Session["lastName"] = user.LastName
	c.Session["role"] = user.Role.String()
	c.Session["eMail"] = user.EMail
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
		//return c.Redirect(App.Activation, loginData.Username) //TODO
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
		c.Validation.Keep()
		return c.Redirect(User.RegistrationPage)
	}

	return c.NotFound("not implemented")
}
