package models

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"turm/app"

	"github.com/revel/revel"
)

/*Salutation is a type for encoding different forms of address. */
type Salutation int

const (
	//NONE is for no form of address
	NONE Salutation = iota
	//MR is for Mr.
	MR
	//MS is for Ms.
	MS
)

func (s Salutation) String() string {
	return [...]string{"none", "mr", "ms"}[s]
}

/*Role is a type for encoding different user roles. */
type Role int

const (
	//USER is the default role without any extra privileges
	USER Role = iota
	//CREATOR allows the creation of courses
	CREATOR
	//ADMIN grants all privileges
	ADMIN
)

func (u Role) String() string {
	return [...]string{"user", "creator", "admin"}[u]
}

/*User contains all directly user related values. */
type User struct {
	ID         int            `db:"id, primarykey, autoincrement"`
	LastName   string         `db:"lastname"`
	FirstName  string         `db:"firstname"`
	EMail      string         `db:"email, unique"`
	Salutation Salutation     `db:"salutation"`
	Role       Role           `db:"role"`
	LastLogin  string         `db:"lastlogin"`
	FirstLogin string         `db:"firstlogin"`
	Language   sql.NullString `db:"language"`

	//ldap user fields
	MatrNr        sql.NullInt32  `db:"matrnr, unique"`
	AcademicTitle sql.NullString `db:"academictitle"`
	Title         sql.NullString `db:"title"`
	NameAffix     sql.NullString `db:"nameaffix"`
	Affiliations  Affiliations   `db:"affiliations"`
	Studies       []Studies      ``

	//external user fields
	Password       sql.NullString `db:"password"`
	PasswordRepeat string         `` //not a field in the respective table
	ActivationCode sql.NullString `db:"activationcode"`
}

func (user *User) String() string {
	return fmt.Sprintf("User(%s)", user.FirstName)
}

/*Validate validates the User struct fields as retrieved by the register form. */
func (user *User) Validate(v *revel.Validation) {

	user.EMail = strings.ToLower(user.EMail)

	v.Check(user.LastName,
		revel.Required{},
		revel.MaxSize{255},
	).MessageKey("validation.invalid.lastname")

	v.Check(user.FirstName,
		revel.Required{},
		revel.MaxSize{255},
	).MessageKey("validation.invalid.firstname")

	v.Check(user.EMail,
		revel.Required{},
		revel.MaxSize{255},
	).MessageKey("validation.invalid.email")

	v.Email(user.EMail).
		MessageKey("validation.invalid.email")

	data := ValidateUniqueData{
		Column: "email",
		Table:  "users",
		Value:  user.EMail,
	}
	v.Check(data, Unique{}).
		MessageKey("validation.email.notUnique")

	isLdapEMail := !strings.Contains(user.EMail, app.EMailSuffix)
	v.Required(isLdapEMail).
		MessageKey("validation.email.ldap")

	v.Check(user.Password.String,
		revel.Required{},
		revel.MaxSize{127},
		revel.MinSize{6},
	).MessageKey("validation.invalid.passwords")

	equal := (user.Password.String == user.PasswordRepeat)
	v.Required(equal).
		MessageKey("validation.invalid.passwords")
	v.Required(user.PasswordRepeat).
		MessageKey("validation.invalid.passwords")

	user.Password.Valid = true

	v.Check(user.Language,
		revel.Required{},
		LanguageValidator{},
	).MessageKey("validation.invalid.language")

	user.Language.Valid = true

	if user.Salutation != NONE && user.Salutation != MR && user.Salutation != MS {
		v.ErrorKey("validation.invalid.salutation")
	}
}

/*Studies contains all data about the course of study of an user. */
type Studies struct {
	UserID            int    `db:"userid, primarykey"`
	Semester          int    `db:"semester"`
	DegreeID          int    `db:"degreeid, primarykey"`
	CourseOfStudiesID int    `db:"courseofstudiesid, primarykey"`
	Degree            string `db:"degree"`          //not a field in the studies table
	CourseOfStudies   string `db:"courseofstudies"` //not a field in the studies table
}

/*Validate validates the Studies struct fields. */
func (studies Studies) Validate(v *revel.Validation) {
	//TODO
}

/*Credentials entered at the login page. */
type Credentials struct {
	Username     string
	EMail        string
	Password     string
	StayLoggedIn bool
}

/*Validate ensures that neither the username nor the password are empty or of incorrect size. */
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

/*Affiliations contains all affiliations of a user. */
type Affiliations []string

/*Value constructs a SQL Value from Affiliations. */
func (affiliations Affiliations) Value() (driver.Value, error) {

	var str string
	for _, affiliation := range affiliations {
		str += `"` + affiliation + `",`
	}
	return driver.Value("{" + strings.TrimRight(str, ",") + "}"), nil
}

/*Scan constructs Affiliations from an SQL Value. */
func (affiliations *Affiliations) Scan(value interface{}) error {

	switch value.(type) {
	case string:
		str := value.(string)
		strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(str, "{", ""), "}", ""))
		*affiliations = strings.Split(str, ",")
	default:
		return errors.New("incompatible type for Affiliations")
	}
	return nil
}
