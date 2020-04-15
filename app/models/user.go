package models

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"

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
	ID         int        `db:"id, primarykey, autoincrement"`
	LastName   string     `db:"lastname"`
	FirstName  string     `db:"firstname"`
	EMail      string     `db:"email, unique"`
	Salutation Salutation `db:"salutation"`
	Role       Role       `db:"role"`
	LastLogin  string     `db:"lastlogin"`
	FirstLogin string     `db:"firstlogin"`

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

/*ValidateUser validates the User struct fields as retrieved by the register form. */
func (user *User) ValidateUser(v *revel.Validation) {

	v.Required(user.LastName).MessageKey("validation.missing.lastname")
	v.Required(user.FirstName).MessageKey("validation.missing.firstname")
	v.Required(user.EMail).MessageKey("validation.missing.email")
	v.Required(user.Password.String).MessageKey("validation.missing.password")
	v.Required(user.PasswordRepeat).MessageKey("validation.missing.passwordRepeat")

	v.MaxSize(user.LastName, 255).MessageKey("validation.max.lastname")
	v.MaxSize(user.FirstName, 255).MessageKey("validation.max.firstname")
	v.MaxSize(user.EMail, 255).MessageKey("validation.max.email")
	v.MaxSize(user.Password.String, 511).MessageKey("validation.max.password")

	v.Email(user.EMail).MessageKey("validation.invalid.email")
	v.Required(user.Password.String == user.PasswordRepeat).MessageKey("validation.invalid.password")
	if user.Salutation != NONE && user.Salutation != MR && user.Salutation != MS {
		v.ErrorKey("validation.invalid.salutation")
	}

	user.Password.Valid = true
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

/*ValidateStudies validates the Studies struct fields. */
func (studies Studies) ValidateStudies(v *revel.Validation) {
	//TODO
}

/*Credentials entered at the login page. */
type Credentials struct {
	Username     string
	EMail        string
	Password     string
	StayLoggedIn bool
}

/*ValidateCredentials ensures that neither the username nor the password are empty or of incorrect size. */
func (credentials *Credentials) ValidateCredentials(v *revel.Validation) {

	if credentials.Username != "" {
		v.MaxSize(credentials.Username, 255).MessageKey("validation.max.username")
		v.Check(credentials.EMail, NotRequiredValidator{}).MessageKey("validation.invalid.credentials")
	} else if credentials.EMail != "" {
		v.MaxSize(credentials.EMail, 255).MessageKey("validation.max.email")
		v.Email(credentials.EMail).MessageKey("validation.invalid.email")
		v.Check(credentials.Username, NotRequiredValidator{}).MessageKey("validation.invalid.credentials")
	} else {
		v.ErrorKey("validation.missing.username")
	}

	v.Required(credentials.Password).MessageKey("validation.missing.password")
	v.MaxSize(credentials.Password, 511).MessageKey("validation.max.password")
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
