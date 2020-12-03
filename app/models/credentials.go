package models

import "github.com/revel/revel"

/*Credentials entered at the login page. */
type Credentials struct {
	Username     string
	EMail        string
	Password     string
	StayLoggedIn bool
}

/*Validate the credentials. */
func (credentials *Credentials) Validate(v *revel.Validation) {

	if credentials.Username != "" { //ldap login credentials

		v.MaxSize(credentials.Username, 255).
			MessageKey("validation.invalid.username")

		v.Check(credentials.EMail,
			NotRequired{},
		).MessageKey("validation.invalid.credentials")

	} else if credentials.EMail != "" { //external login credentials

		v.Required(credentials.EMail).
			MessageKey("validation.invalid.email")

		v.Email(credentials.EMail).
			MessageKey("validation.invalid.email")

		v.Check(credentials.Username,
			NotRequired{},
		).MessageKey("validation.invalid.credentials")

	} else { //neither username nor e-mail address was provided
		v.ErrorKey("validation.invalid.username")
	}

	v.Check(credentials.Password,
		revel.Required{},
		revel.MaxSize{127},
	).MessageKey("validation.invalid.password")
}
