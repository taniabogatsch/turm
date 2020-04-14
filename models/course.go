package models

import (
	"database/sql"

	"github.com/revel/revel"
)

/*Course contains all directly course related values. */
type Course struct {
	ID                int             `db:"id, primarykey, autoincrement"`
	Title             string          `db:"title"`
	Creator           sql.NullInt32   `db:"creator"`
	Subtitle          sql.NullString  `db:"subtitle"`
	Visible           bool            `db:"visible"`
	Active            bool            `db:"active"`
	OnlyLDAP          bool            `db:"onlyldap"`
	CreationDate      string          `db:"creationdate"`
	Description       sql.NullString  `db:"description"`
	Fee               sql.NullFloat64 `db:"fee"`
	CustomEMail       sql.NullString  `db:"customemail"`
	EnrollLimitEvents sql.NullInt32   `db:"enrolllimitevents"`
	EnrollmentStart   string          `db:"enrollmentstart"`
	EnrollmentEnd     string          `db:"enrollmentend"`
	UnsubscribeEnd    sql.NullString  `db:"unsubscribeend"`
	ExpirationDate    string          `db:"expirationdate"`
	Events            []Event         ``
	Editors           []UserList      ``
	Instructors       []UserList      ``
	Blacklist         []UserList      ``
	Whitelist         []UserList      ``
	Restrictions      []Restriction   ``
}

/*ValidateCourse validates the Course struct fields. */
func (course *Course) ValidateCourse(v *revel.Validation) {
	//TODO
}

/*UserList contains all users that are in one of the user lists of a course,
which are: editors, instructors, blacklist, whitelist. */
type UserList struct {
	UserID     int    `db:"userid, primarykey"`
	CourseID   int    `db:"courseid, primarykey"`
	ViewMatrNr bool   `db:"viewmatrnr"` //only a field in the tables editor and instructor
	LastName   string `db:"lastname"`   //not a field in the respective table
	FirstName  string `db:"firstname"`  //not a field in the respective table
	EMail      string `db:"email"`      //not a field in the respective table
}

//validateUserList validates the UserList struct fields.
func (user *UserList) validateUserList(v *revel.Validation) {
	//TODO
}

/*Restriction contains all data about an enrollment restriction of a course. */
type Restriction struct {
	ID                int `db:"id, primarykey, autoincrement"`
	CourseID          int `db:"courseid"`
	MinimumSemester   int `db:"minimumsemester"`
	DegreeID          int `db:"degreeid"`
	CourseOfStudiesID int `db:"courseofstudiesid"`
}

//validateRestriction validates the Restriction struct fields.
func (restriction *Restriction) validateRestriction(v *revel.Validation) {
	//TODO
}
