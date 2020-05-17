package models

import (
	"turm/app"
)

/*CourseListInfo holds only the most essential information about courses. */
type CourseListInfo struct {
	ID           int    `db:"id, primarykey, autoincrement"`
	Title        string `db:"title"`
	CreationDate string `db:"creation_date"`
	EMail        string `db:"email"` //e-mail address of either the creator or the editor
}

/*CourseList holds the most essential information about a list of courses. */
type CourseList []CourseListInfo

/*GetByUserID returns all courses according to the user type.  */
func (list *CourseList) GetByUserID(userID *int, userType string, active, expired bool) (err error) {

	//construct SQL
	stmtSelect := `
		SELECT c.id, c.title, u.email,
			TO_CHAR (c.creation_date AT TIME ZONE $1, 'YYYY-MM-DD HH24:MI') as creation_date
	`
	stmtWhere := `
			AND c.active = $3
			AND (current_timestamp >= c.expiration_date) = $4
		ORDER BY c.creation_date DESC
	`
	if !expired && !active {
		stmtWhere = `
				AND c.active = $3
				AND (
					(current_timestamp < c.expiration_date) = $4
					OR
					(current_timestamp >= c.expiration_date) = $4
				)
			ORDER BY c.creation_date DESC
		`
	}
	stmt := ``
	if userType == "creator" { //get all created courses
		stmt = `
		 	FROM courses c, users u
		 	WHERE c.creator = u.id
		 		AND u.id = $2
		`
	} else { //get all edit/instruct privilege courses
		stmt = `
			FROM courses c, users u, ` + userType + ` l
			WHERE c.id = l.course_id
				AND u.id = $2
				AND u.id = l.user_id
		`
	}

	if userType == "admin" {
		err = app.Db.Select(list, stmtAllCoursesAdmin+stmtWhere, app.TimeZone, *userID, active, expired)
	} else {
		err = app.Db.Select(list, stmtSelect+stmt+stmtWhere, app.TimeZone, *userID, active, expired)
	}

	if err != nil {
		log.Error("failed to get course list", "user ID", *userID,
			"userType", userType, "active", active, "expired", expired,
			"error", err.Error())
	}
	return
}

const (
	stmtAllCoursesAdmin = `
		SELECT c.id, c.title,
			TO_CHAR (c.creation_date AT TIME ZONE $1, 'YYYY-MM-DD HH24:MI') as creation_date
		FROM courses c
		WHERE $2 = 0
	`
)
