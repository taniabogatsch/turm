package models

import (
	"database/sql"
	"turm/app"

	"github.com/jmoiron/sqlx"
	"github.com/revel/revel"
)

/*Restrictions of a course. */
type Restrictions []Restriction

/*Restriction is a model of the restriction table. */
type Restriction struct {
	ID                int           `db:"id, primarykey, autoincrement"`
	CourseID          int           `db:"course_id"`
	MinimumSemester   sql.NullInt64 `db:"minimum_semester"`
	DegreeID          sql.NullInt64 `db:"degree_id"`
	CourseOfStudiesID sql.NullInt64 `db:"courses_of_studies_id"`

	//for usability
	DegreeName  sql.NullString `db:"degree_name"`
	StudiesName sql.NullString `db:"studies_name"`
}

/*CoursesOfStudies holds all existing courses of studies. */
type CoursesOfStudies []CourseOfStudies

/*CourseOfStudies of a student. */
type CourseOfStudies struct {
	ID   int    `db:"id, primarykey, autoincrement"`
	Name string `db:"name"`
}

/*Degrees holds all existing degrees. */
type Degrees []Degree

/*Degree pursued by a student. */
type Degree struct {
	ID   int    `db:"id, primarykey, autoincrement"`
	Name string `db:"name"`
}

/*Get all restrictions of a course. */
func (rests *Restrictions) Get(tx *sqlx.Tx, courseID *int) (err error) {

	if tx == nil {
		err = app.Db.Select(rests, stmtSelectRestrictions, *courseID)
	} else {
		err = tx.Select(rests, stmtSelectRestrictions, *courseID)
	}

	if err != nil {
		log.Error("failed to get restrictions", "courseID", *courseID,
			"error", err.Error())
		if tx != nil {
			tx.Rollback()
		}
	}
	return
}

/*Duplicate all restrictions of a course. */
func (rests *Restrictions) Duplicate(tx *sqlx.Tx, courseID, courseIDOld *int) (err error) {

	if _, err = tx.Exec(stmtDuplicateRestrictions, *courseID, *courseIDOld); err != nil {
		log.Error("failed to duplicate restrictions", "courseID", *courseID, "courseIDOld",
			*courseIDOld, "error", err.Error())
		tx.Rollback()
	}

	return
}

/*Validate Restriction fields. */
func (rest *Restriction) Validate(v *revel.Validation) {

	if rest.CourseOfStudiesID.Int64 != 0 {
		rest.CourseOfStudiesID.Valid = true
	}
	if rest.DegreeID.Int64 != 0 {
		rest.DegreeID.Valid = true
	}
	if rest.MinimumSemester.Int64 != 0 {
		rest.MinimumSemester.Valid = true
	}

	if !rest.CourseOfStudiesID.Valid && !rest.DegreeID.Valid &&
		!rest.MinimumSemester.Valid {
		v.ErrorKey("validation.invalid.restriction")
	}
}

/*Insert restriction. */
func (rest *Restriction) Insert(tx *sqlx.Tx, courseID int) (err error) {

	if courseID != 0 {
		rest.CourseID = courseID
	}

	if tx == nil {
		err = app.Db.Get(rest, stmtInsertRestriction, rest.CourseID, rest.MinimumSemester,
			rest.DegreeID, rest.CourseOfStudiesID)
	} else {
		err = tx.Get(rest, stmtInsertRestriction, rest.CourseID, rest.MinimumSemester,
			rest.DegreeID, rest.CourseOfStudiesID)
	}

	if err != nil {
		log.Error("failed to insert restriction", "restriction", *rest,
			"error", err.Error())
		if tx != nil {
			tx.Rollback()
		}
	}
	return
}

/*Update restriction. */
func (rest *Restriction) Update() (err error) {

	err = app.Db.Get(rest, stmtUpdateRestriction, rest.MinimumSemester,
		rest.DegreeID, rest.CourseOfStudiesID, rest.ID)
	if err != nil {
		log.Error("failed to update restriction", "restriction", *rest,
			"error", err.Error())
	}
	return
}

/*Delete a restriction. */
func (rest *Restriction) Delete() (err error) {

	_, err = app.Db.Exec(stmtDeleteRestriction, rest.ID)
	if err != nil {
		log.Error("failed to delete restriction", "restriction", *rest,
			"error", err.Error())
	}
	return
}

/*Get all courses of studies. */
func (courses *CoursesOfStudies) Get(tx *sqlx.Tx) (err error) {

	if err = tx.Select(courses, stmtSelectCoursesOfStudies); err != nil {
		log.Error("failed to get courses of studies", "error", err.Error())
		tx.Rollback()
	}
	return
}

/*Get all degrees. */
func (degrees *Degrees) Get(tx *sqlx.Tx) (err error) {

	if err = tx.Select(degrees, stmtSelectDegrees); err != nil {
		log.Error("failed to get degrees", "error", err.Error())
		tx.Rollback()
	}
	return
}

const (
	stmtSelectRestrictions = `
		SELECT r.id, r.course_id, r.minimum_semester, r.degree_id,
			r.courses_of_studies_id, d.name AS degree_name, s.name AS studies_name
		FROM enrollment_restrictions r LEFT OUTER JOIN
			degrees d ON r.degree_id = d.id LEFT OUTER JOIN
			courses_of_studies s ON r.courses_of_studies_id = s.id
		WHERE r.course_id = $1
		ORDER BY
			studies_name ASC, degree_name ASC
	`

	stmtInsertRestriction = `
		INSERT INTO enrollment_restrictions
			(course_id, minimum_semester, degree_id, courses_of_studies_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	stmtUpdateRestriction = `
		UPDATE enrollment_restrictions
		SET minimum_semester = $1, degree_id = $2, courses_of_studies_id = $3
		WHERE id = $4
		RETURNING id
	`

	stmtDeleteRestriction = `
		DELETE FROM enrollment_restrictions
		WHERE id = $1
	`

	stmtSelectCoursesOfStudies = `
		SELECT id, name
		FROM courses_of_studies
		ORDER BY name ASC
	`

	stmtSelectDegrees = `
		SELECT id, name
		FROM degrees
		ORDER BY name ASC
	`

	stmtDuplicateRestrictions = `
		INSERT INTO enrollment_restrictions
			(course_id, minimum_semester, degree_id, courses_of_studies_id)
		(
			SELECT $1 AS course_id, minimum_semester, degree_id,
				courses_of_studies_id
			FROM enrollment_restrictions
			WHERE course_id = $2
		)
	`
)
