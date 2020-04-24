package models

import "github.com/revel/revel"

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

/*Validate validates the UserList struct fields. */
func (user *UserList) Validate(v *revel.Validation) {
	//TODO
}
