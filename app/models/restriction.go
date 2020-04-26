package models

import "github.com/revel/revel"

/*Restriction is a model of the restriction table. */
type Restriction struct {
	ID                int `db:"id, primarykey, autoincrement"`
	CourseID          int `db:"courseid"`
	MinimumSemester   int `db:"minimumsemester"`
	DegreeID          int `db:"degreeid"`
	CourseOfStudiesID int `db:"courseofstudiesid"`
}

/*Validate Restriction fields. */
func (restriction *Restriction) Validate(v *revel.Validation) {
	//TODO
}
