package models

import "github.com/revel/revel"

/*Restriction is a model of the restriction table. */
type Restriction struct {
	ID                int `db:"id, primarykey, autoincrement"`
	CourseID          int `db:"course_id"`
	MinimumSemester   int `db:"minimum_semester"`
	DegreeID          int `db:"degree_id"`
	CourseOfStudiesID int `db:"courses_of_studies_id"`
}

/*Validate Restriction fields. */
func (restriction *Restriction) Validate(v *revel.Validation) {
	//TODO
}
