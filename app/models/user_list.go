package models

import (
	"database/sql"
	"strconv"
	"strings"
	"turm/app"

	"github.com/jmoiron/sqlx"
)

/*UserListEntry is a model of the user list tables,
which are: editors, instructors, blocklist, allowlist.
It is also used to render users for the different user
searches at the course management page. */
type UserListEntry struct {
	UserID     int  `db:"user_id, primarykey"`
	CourseID   int  `db:"course_id, primarykey"`
	ViewMatrNr bool `db:"view_matr_nr"` //only a field in the tables editor and instructor

	//identifies whether a user is already on a user list
	OnList bool `db:"on_list"`

	//used for showing users at user searches
	EMail string `db:"email, unique"`

	//used for showing users properly
	AcademicTitle  sql.NullString `db:"academic_title"`
	Title          sql.NullString `db:"title"`
	NameAffix      sql.NullString `db:"name_affix"`
	LastName       string         `db:"last_name"`
	FirstName      string         `db:"first_name"`
	Salutation     Salutation     `db:"salutation"`
	ActivationCode sql.NullString `db:"activation_code"`
}

/*Insert the provided user list entry of a course. */
func (user *UserListEntry) Insert(table string) (active bool, data EMailData, err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	//construct SQL
	colViewMatrNr := ""
	colViewMatrNrValue := ""
	if table == TableEditors || table == TableInstructors {
		colViewMatrNr = ", view_matr_nr"
		colViewMatrNrValue = ", $3"
	}

	insertUser := `
		INSERT INTO ` + table + `
			(user_id, course_id` + colViewMatrNr + `)
		VALUES ($1, $2` + colViewMatrNrValue + `)
		RETURNING (
			SELECT email
			FROM users
			WHERE id = $1
		), course_id
	`

	if table == TableEditors || table == TableInstructors {

		//get if the course is active
		course := Course{ID: user.CourseID}
		if err = course.GetColumnValue(tx, "active"); err != nil {
			return
		}
		active = course.Active

		if active {

			//get missing e-mail data
			if err = course.GetColumnValue(tx, "title"); err != nil {
				return
			}

			//set e-mail data
			data.CourseTitle = course.Title
			data.CourseID = course.ID
			data.User.ID = user.UserID
			if err = data.User.Get(tx); err != nil {
				return
			}

			data.CourseRole = table
			data.ViewMatrNr = true
		}

		//insert user
		err = tx.Get(user, insertUser, user.UserID, user.CourseID, true)

	} else {

		//insert user
		err = tx.Get(user, insertUser, user.UserID, user.CourseID)
	}

	if err != nil {
		log.Error("failed to insert user into user list", "user", user,
			"table", table, "error", err.Error())
		tx.Rollback()
		return
	}

	tx.Commit()
	return
}

/*Delete the provided user list entry of a course. */
func (user *UserListEntry) Delete(table string) (active bool, data EMailData, err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	deleteUser := `
		DELETE FROM ` + table + `
		WHERE user_id = $1
			AND course_id = $2
	`

	_, err = tx.Exec(deleteUser, user.UserID, user.CourseID)
	if err != nil {
		log.Error("failed to delete user from user list", "user", user,
			"table", table, "error", err.Error())
		tx.Rollback()
		return
	}

	if table == "editors" || table == "instructors" {

		//get if the course is active
		course := Course{ID: user.CourseID}
		if err = course.GetColumnValue(tx, "active"); err != nil {
			return
		}
		active = course.Active

		if active {

			//get missing e-mail data
			if err = course.GetColumnValue(tx, "title"); err != nil {
				return
			}

			//set e-mail data
			data.CourseTitle = course.Title
			data.CourseID = course.ID
			data.User.ID = user.UserID
			if err = data.User.Get(tx); err != nil {
				return
			}

			data.CourseRole = table
		}
	}

	tx.Commit()
	return
}

/*Update updates the ViewMatrNr field of a list entry of a course. */
func (user *UserListEntry) Update(table string) (active bool, data EMailData, err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	updateUser := `
		UPDATE ` + table + `
		SET view_matr_nr = $3
		WHERE user_id = $1
			AND course_id = $2
		RETURNING (
			SELECT email
			FROM users
			WHERE id = $1
		), course_id
	`

	err = tx.Get(user, updateUser, user.UserID, user.CourseID, user.ViewMatrNr)
	if err != nil {
		log.Error("failed to update user from user list", "user", user,
			"table", table, "error", err.Error())
	}

	if table == "editors" || table == "instructors" {

		//get if the course is active
		course := Course{ID: user.CourseID}
		if err = course.GetColumnValue(tx, "active"); err != nil {
			return
		}
		active = course.Active

		if active {

			//get missing e-mail data
			if err = course.GetColumnValue(tx, "title"); err != nil {
				return
			}

			//set e-mail data
			data.CourseTitle = course.Title
			data.CourseID = course.ID
			data.User.ID = user.UserID
			if err = data.User.Get(tx); err != nil {
				return
			}

			data.ViewMatrNr = user.ViewMatrNr
		}
	}

	tx.Commit()
	return
}

/*Exists returns if a user exists in the DB. */
func (user *UserListEntry) Exists(tx *sqlx.Tx) (exists bool, err error) {

	err = tx.Get(&exists, stmtUserExists, user.UserID)
	if err != nil {
		log.Error("failed to get if the user exists", "userID", user.UserID,
			"error", err.Error())
		tx.Rollback()
	}
	return
}

/*UserList holds users enlisted on one (or more) of the user lists. */
type UserList []UserListEntry

/*Get all users at a user list of a course. */
func (users *UserList) Get(tx *sqlx.Tx, courseID *int, table string) (err error) {

	//construct SQL
	colViewMatrNr := ""
	if table == TableEditors || table == TableInstructors {
		colViewMatrNr = "l.view_matr_nr,"
	}
	selectUsers := `
		SELECT
			l.user_id, l.course_id, ` + colViewMatrNr + `
			u.first_name, u.last_name, u.email, u.salutation,
			u.title, u.academic_title, u.name_affix
		FROM ` + table + ` l, users u
		WHERE l.user_id = u.id
			AND l.course_id = $1
	`

	if tx == nil {
		err = app.Db.Select(users, selectUsers, *courseID)
	} else {
		err = tx.Select(users, selectUsers, *courseID)
	}
	if err != nil {
		log.Error("failed to get user list", "table", table, "course ID",
			*courseID, "error", err.Error())
		if tx != nil {
			tx.Rollback()
		}
	}
	return
}

/*Duplicate the user list of a course. */
func (users *UserList) Duplicate(tx *sqlx.Tx, courseIDNew, courseIDOld *int, table string) (err error) {

	//construct SQL
	colViewMatrNr := ""
	if table == TableEditors || table == TableInstructors {
		colViewMatrNr = ", view_matr_nr"
	}
	stmtDuplicateList := `
		INSERT INTO ` + table + `
			(course_id, user_id` + colViewMatrNr + `)
		(
			SELECT $2 AS course_id, user_id` + colViewMatrNr + `
			FROM ` + table + `
			WHERE course_id = $1
		)
	`

	_, err = tx.Exec(stmtDuplicateList, *courseIDOld, *courseIDNew)
	if err != nil {
		log.Error("failed to duplicate user list", "table", table, "course ID old",
			*courseIDOld, "course ID new", *courseIDNew, "error", err.Error())
		tx.Rollback()
	}
	return
}

/*Search for a user and identify whether that user is already on a user list. */
func (users *UserList) Search(value, listType *string, searchInactive *bool, courseID *int) (err error) {

	searchUsersSelect := `
		SELECT
			id as user_id, email, activation_code,
			(
				SELECT EXISTS (
					SELECT true
					FROM ` + *listType + ` t
					WHERE t.user_id = u.id
						AND t.course_id = $4
				)
			) AS on_list
	`
	stmt := searchUsersSelect + " " + stmtUsersWhere

	//prepare the search value for pattern matching
	values := "%"
	for _, val := range strings.Split(*value, " ") {
		values += val + "%"
	}
	//the value can be the matriculation number
	matrNr, _ := strconv.Atoi(*value) //matrNr is 0 if there is an error

	err = app.Db.Select(users, stmt, values, matrNr, *searchInactive, *courseID)
	if err != nil {
		log.Error("failed to search users", "values", values,
			"matrNr", matrNr, "error", err.Error())
	}
	return
}

/*InsertUploaded inserts all entries in a user list into a course. Skips all users with
an invalid user ID. */
func (users *UserList) InsertUploaded(tx *sqlx.Tx, courseID *int, table string) (err error) {

	stmt := `
		INSERT INTO ` + table + `
			(user_id, course_id, view_matr_nr)
		VALUES ($1, $2, $3)
	`
	if table != TableEditors && table != TableInstructors {
		stmt = `
			INSERT INTO ` + table + `
				(user_id, course_id)
			VALUES ($1, $2)
		`
	}

	for _, user := range *users {

		//only try to insert existing users
		if exists, err := user.Exists(tx); err != nil {
			return err
		} else if !exists {
			continue
		}

		if table != TableEditors && table != TableInstructors {
			_, err = tx.Exec(stmt, user.UserID, *courseID)
		} else {
			_, err = tx.Exec(stmt, user.UserID, *courseID, user.ViewMatrNr)
		}
		if err != nil {
			log.Error("failed to insert user", "table", table, "course ID",
				*courseID, "user", user, "error", err.Error())
			tx.Rollback()
			return
		}
	}

	return
}

/*GetEditorsInstructors returns editors and instructors. */
func (users *UserList) GetEditorsInstructors(courseID *int) (instructors UserList, err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	if err = users.Get(tx, courseID, "editors"); err != nil {
		return
	}
	if err = instructors.Get(tx, courseID, "instructors"); err != nil {
		return
	}

	tx.Commit()
	return
}

const (
	stmtUsersWhere = `
		FROM users u
		WHERE (
				(u.activation_code IS NOT NULL) = $3
				OR u.activation_code IS NULL
			)
			AND (
				/* all combinations having a name_affix */
				u.title || u.academic_title || u.first_name || u.name_affix || u.last_name ILIKE $1
				OR u.title || u.first_name || u.name_affix || u.last_name ILIKE $1
				OR u.academic_title || u.first_name || u.name_affix || u.last_name ILIKE $1
				OR u.first_name || u.name_affix || u.last_name ILIKE $1

				/* all combinations without a name_affix */
				OR u.title || u.academic_title || u.first_name || u.last_name ILIKE $1
				OR u.title || u.first_name || u.last_name ILIKE $1
				OR u.academic_title || u.first_name || u.last_name ILIKE $1
				OR u.first_name || u.last_name ILIKE $1

				/* others */
				OR u.email ILIKE $1
				OR u.matr_nr = $2
			)
		ORDER BY u.last_name, u.first_name, u.id
		LIMIT 5
	`

	stmtUserExists = `
		SELECT EXISTS (
			SELECT id
			FROM users
			WHERE id = $1
		) AS exists
	`
)
