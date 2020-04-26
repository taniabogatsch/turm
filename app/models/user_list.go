package models

import (
	"database/sql"
	"strconv"
	"strings"
	"turm/app"

	"github.com/jmoiron/sqlx"
)

/*UserListEntry is a model of the user list tables,
which are: editors, instructors, blacklist, whitelist.
It is also used to render users for the different user
searches at the course management page. */
type UserListEntry struct {
	UserID     int  `db:"userid, primarykey"`
	CourseID   int  `db:"courseid, primarykey"`
	ViewMatrNr bool `db:"viewmatrnr"` //only a field in the tables editor and instructor

	//identifies whether a user is already on a user list
	OnList bool `db:"onlist"`

	//used for showing users at user searches
	EMail string `db:"email, unique"`

	//used for showing users properly
	AcademicTitle sql.NullString `db:"academictitle"`
	Title         sql.NullString `db:"title"`
	NameAffix     sql.NullString `db:"nameaffix"`
	LastName      string         `db:"lastname"`
	FirstName     string         `db:"firstname"`
	Salutation    Salutation     `db:"salutation"`
}

/*Insert the provided user list entry of a course. */
func (user *UserListEntry) Insert(table string) (err error) {

	//construct SQL
	colViewMatrNr := ""
	colViewMatrNrValue := ""
	if table == "editor" || table == "instructor" {
		colViewMatrNr = ", viewmatrnr"
		colViewMatrNrValue = ", $3"
	}

	insertUser := `
		INSERT INTO ` + table + `
			(userid, courseid` + colViewMatrNr + `)
		VALUES ($1, $2` + colViewMatrNrValue + `)
		RETURNING (
			SELECT email
			FROM users
			WHERE id = $1
		)
	`

	if table == "editor" || table == "instructor" {
		err = app.Db.Get(user, insertUser, user.UserID, user.CourseID, true)
	} else {
		err = app.Db.Get(user, insertUser, user.UserID, user.CourseID)
	}
	if err != nil {
		modelsLog.Error("failed to insert user into user list", "user", user, "error", err.Error())
	}
	return
}

/*UserList holds users enlisted on one (or more) of the user lists. */
type UserList []UserListEntry

/*Get all users at a user list of a course. */
func (users *UserList) Get(tx *sqlx.Tx, courseID *int, table string) (err error) {

	//construct SQL
	colViewMatrNr := ""
	if table == "editor" || table == "instructor" {
		colViewMatrNr = "l.viewmatrnr,"
	}
	selectUsers := `
		SELECT
			l.userid, l.courseid, ` + colViewMatrNr + `
			u.firstname, u.lastname, u.email, u.salutation,
			u.title, u.academictitle, u.nameaffix
		FROM ` + table + ` l, users u
		WHERE l.userid = u.id
			AND l.courseid = $1
	`
	err = tx.Select(users, selectUsers, *courseID)
	if err != nil {
		modelsLog.Error("failed to get user list", "table", table, "course ID",
			*courseID, "error", err.Error())
		tx.Rollback()
	}
	return
}

/*Search for a user and identify whether that user is already on a user list. */
func (users *UserList) Search(value *string, searchInactive *bool, listType *string) (err error) {

	searchUsersSelect := `
		SELECT
			id as userid, email,
			(
				SELECT EXISTS (
					SELECT true
					FROM ` + *listType + ` t, users us
					WHERE t.userid = us.id
						AND us.id = u.id
				)
			) AS onlist
	`
	stmt := searchUsersSelect + " " + stmtUsersWhere

	//prepare the search value for pattern matching
	values := "%"
	for _, val := range strings.Split(*value, " ") {
		values += val + "%"
	}
	//the value can be the matriculation number
	matrNr, _ := strconv.Atoi(*value) //matrNr is 0 if there is an error

	err = app.Db.Select(users, stmt, values, matrNr, *searchInactive)
	if err != nil {
		modelsLog.Error("failed to search users", "values", values,
			"matrNr", matrNr, "error", err.Error())
	}
	return
}

const (
	stmtUsersWhere = `
		FROM users u
		WHERE (
				(u.activationcode IS NOT NULL) = $3
				OR u.activationcode IS NULL
			)
			AND (
				/* all combinations having a nameaffix */
				u.title || u.academictitle || u.firstname || u.nameaffix || u.lastname ILIKE $1
				OR u.title || u.firstname || u.nameaffix || u.lastname ILIKE $1
				OR u.academictitle || u.firstname || u.nameaffix || u.lastname ILIKE $1
				OR u.firstname || u.nameaffix || u.lastname ILIKE $1

				/* all combinations without a nameaffix */
				OR u.title || u.academictitle || u.firstname || u.lastname ILIKE $1
				OR u.title || u.firstname || u.lastname ILIKE $1
				OR u.academictitle || u.firstname || u.lastname ILIKE $1
				OR u.firstname || u.lastname ILIKE $1

				/* others */
				OR u.email ILIKE $1
				OR u.matrnr = $2
			)
		ORDER BY u.lastname, u.firstname, u.id
		LIMIT 5
	`
)
