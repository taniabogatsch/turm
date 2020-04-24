package models

import "github.com/revel/revel"

/*Restriction contains all data about an enrollment restriction of a course. */
type Restriction struct {
	ID                int `db:"id, primarykey, autoincrement"`
	CourseID          int `db:"courseid"`
	MinimumSemester   int `db:"minimumsemester"`
	DegreeID          int `db:"degreeid"`
	CourseOfStudiesID int `db:"courseofstudiesid"`
}

/*Validate validates the Restriction struct fields. */
func (restriction *Restriction) Validate(v *revel.Validation) {
	//TODO
}
