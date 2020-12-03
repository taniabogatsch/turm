package models

import (
	"github.com/jmoiron/sqlx"
	"github.com/revel/revel"
)

/*Studies holds all courses of study of an user. */
type Studies []Study

/*Study is a model of the studies table. */
type Study struct {
	UserID            int    `db:"user_id, primarykey"`
	Semester          int    `db:"semester"`
	DegreeID          int    `db:"degree_id, primarykey"`
	CourseOfStudiesID int    `db:"course_of_studies_id, primarykey"`
	Degree            string `db:"degree"`            //not a field in the studies table
	CourseOfStudies   string `db:"course_of_studies"` //not a field in the studies table
}

/*Validate Studies fields when loaded from the user enrollment file. */
func (studies *Study) Validate(v *revel.Validation) {
	//TODO
}

/*Select all courses of studies of a user. */
func (studies *Studies) Select(tx *sqlx.Tx, userID *int) (err error) {

	err = tx.Select(studies, stmtSelectUserCoursesOfStudies, *userID)
	if err != nil {
		log.Error("failed to get courses of studies of user", "userID", *userID,
			"error", err.Error())
		tx.Rollback()
	}
	return
}
