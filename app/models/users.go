package models

import (
	"strconv"
	"strings"
	"turm/app"

	"github.com/revel/revel"
)

/*Users contains specific users, such as only admins, creators, ... */
type Users []User

/*Get specific users. */
func (users *Users) Get(role Role) (err error) {

	selectUsers := `
		SELECT id, lastname, firstname, email, salutation,
			TO_CHAR (lastlogin AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') as lastlogin
		FROM users
		WHERE role = $1
			AND activationcode IS NULL
		ORDER BY lastname, firstname, id
	`
	switch role {
	case ADMIN:
		err = app.Db.Select(users, selectUsers, ADMIN, app.TimeZone)
		if err != nil {
			modelsLog.Error("failed to get users", "role", ADMIN, "error", err.Error())
		}
	case CREATOR:
		err = app.Db.Select(users, selectUsers, CREATOR, app.TimeZone)
		if err != nil {
			modelsLog.Error("failed to get users", "role", CREATOR, "error", err.Error())
		}
	default:
		modelsLog.Error("invalid role provided", "role", role)
	}

	return
}

/*Search for users. */
func (users *Users) Search(value *string, searchInactive *bool) (err error) {

	//prepare the search value for pattern matching
	values := "%"
	for _, val := range strings.Split(*value, " ") {
		values += val + "%"
	}
	//the value can be the matriculation number
	matrNr, _ := strconv.Atoi(*value) //matrNr is 0 if there is an error

	stmt := stmtSearchUsers
	if *searchInactive {
		stmt = stmtSearchUsersIncludeInactive
	}

	err = app.Db.Select(users, stmt, values, matrNr, app.TimeZone)
	if err != nil {
		modelsLog.Error("failed to search users", "values", values,
			"matrNr", matrNr, "error", err.Error())
	}
	return
}

/*Studies is a model of the studies table. */
type Studies struct {
	UserID            int    `db:"userid, primarykey"`
	Semester          int    `db:"semester"`
	DegreeID          int    `db:"degreeid, primarykey"`
	CourseOfStudiesID int    `db:"courseofstudiesid, primarykey"`
	Degree            string `db:"degree"`          //not a field in the studies table
	CourseOfStudies   string `db:"courseofstudies"` //not a field in the studies table
}

/*Validate the studies when loaded from the user enrollment file. */
func (studies Studies) Validate(v *revel.Validation) {
	//TODO
}

const (
	stmtSearchUsers = `
    SELECT id, lastname, firstname, email, salutation, role,
      TO_CHAR (lastlogin AT TIME ZONE $3, 'YYYY-MM-DD HH24:MI') as lastlogin
    FROM users
    WHERE activationcode IS NULL
      AND (
        /* all combinations having a nameaffix */
        title || academictitle || firstname || nameaffix || lastname ILIKE $1
        OR title || firstname || nameaffix || lastname ILIKE $1
        OR academictitle || firstname || nameaffix || lastname ILIKE $1
        OR firstname || nameaffix || lastname ILIKE $1

        /* all combinations without a nameaffix */
        OR title || academictitle || firstname || lastname ILIKE $1
        OR title || firstname || lastname ILIKE $1
        OR academictitle || firstname || lastname ILIKE $1
        OR firstname || lastname ILIKE $1

        /* others */
        OR email ILIKE $1
        OR matrnr = $2
      )
    ORDER BY lastname, firstname, id
    LIMIT 5
  `

	stmtSearchUsersIncludeInactive = `
		SELECT id, lastname, firstname, email, salutation, role,
			TO_CHAR (lastlogin AT TIME ZONE $3, 'YYYY-MM-DD HH24:MI') as lastlogin
		FROM users
		WHERE
			/* all combinations having a nameaffix */
			title || academictitle || firstname || nameaffix || lastname ILIKE $1
			OR title || firstname || nameaffix || lastname ILIKE $1
			OR academictitle || firstname || nameaffix || lastname ILIKE $1
			OR firstname || nameaffix || lastname ILIKE $1

			/* all combinations without a nameaffix */
			OR title || academictitle || firstname || lastname ILIKE $1
			OR title || firstname || lastname ILIKE $1
			OR academictitle || firstname || lastname ILIKE $1
			OR firstname || lastname ILIKE $1

			/* others */
			OR email ILIKE $1
			OR matrnr = $2
		ORDER BY lastname, firstname, id
		LIMIT 5
	`
)
