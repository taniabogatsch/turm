package models

import (
	"strings"
	"turm/app"

	"github.com/jmoiron/sqlx"
)

/*Courses holds different courses. */
type Courses []Course

/*CourseListInfo holds only the most essential information about courses. */
type CourseListInfo struct {
	ID           int    `db:"id, primarykey, autoincrement"`
	Title        string `db:"title"`
	CreationDate string `db:"creation_date"`
	EMail        string `db:"email"` //e-mail address of either the creator or the editor
}

/*CourseList holds the most essential information about a list of courses. */
type CourseList []CourseListInfo

/*Search all courses. */
func (list *CourseList) Search(value string) (err error) {

	//we need to divide the string into substrings so we don't have whitespaces
	strSlice := strings.Split(value, " ")
	searchVal := ""
	for _, str := range strSlice {
		searchVal += "%" + str + "%"
	}

	err = app.Db.Select(list, stmtSearchCourses, searchVal)
	if err != nil {
		log.Error("failed to search courses", "value", value, "searchVal",
			searchVal, "error", err.Error())
	}

	return
}

/*SearchForDrafts returns all courses that can be duplicated by a creator. */
func (list *CourseList) SearchForDrafts(value string, userID int, role string) (err error) {

	//we need to divide the string into substrings so we don't have whitespaces
	strSlice := strings.Split(value, " ")
	searchVal := ""
	for _, str := range strSlice {
		searchVal += "%" + str + "%"
	}

	if role == ADMIN.String() {
		err = app.Db.Select(list, stmtSearchCoursesForDraftAdmin, searchVal)
	} else {
		err = app.Db.Select(list, stmtSearchCoursesForDraft, searchVal)
	}

	if err != nil {
		log.Error("failed to search courses", "value", value, "searchVal",
			searchVal, "error", err.Error())
	}

	return
}

/*GetByUserID returns all courses according to the user type.  */
func (list *CourseList) GetByUserID(tx *sqlx.Tx, userID int, userType string, active,
	expired bool) (err error) {

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
		stmt = stmtAllCoursesAdmin + stmtWhere
		userID = 0
	} else {
		stmt = stmtSelect + stmt + stmtWhere
	}

	err = tx.Select(list, stmt, app.TimeZone, userID, active, expired)
	if err != nil {
		log.Error("failed to get course list", "userID", userID, "userType", userType,
			"active", active, "expired", expired, "stmt", stmt, "error", err.Error())
		tx.Rollback()
	}
	return
}

/*Get the course lists for the specified users. */
func (list *CourseList) Get(active, expired bool, userID int, role string) (
	editor, instructor CourseList, err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	//if the user is an admin, render all active courses
	if role == ADMIN.String() {

		err = list.GetByUserID(tx, userID, "admin", active, expired)
		if err != nil {
			return
		}

		tx.Commit()
		return
	}

	err = list.GetByUserID(tx, userID, "creator", active, expired)
	if err != nil {
		return
	}

	err = editor.GetByUserID(tx, userID, "editors", active, expired)
	if err != nil {
		return
	}

	if active { //instructors are not part of drafts
		err = instructor.GetByUserID(tx, userID, "instructors", active, expired)
		if err != nil {
			return
		}
	}

	tx.Commit()
	return
}

const (
	stmtSearchCourses = `
    /* search course fields */
    SELECT c.id, c.title
    FROM courses c LEFT OUTER JOIN events e ON c.id = e.course_id
      LEFT OUTER JOIN meetings m ON e.id = m.event_id
    WHERE
      c.active
      AND c.expiration_date > now()
      AND (
        c.title ILIKE $1
        OR c.description ILIKE $1
        OR c.speaker ILIKE $1
        OR c.subtitle ILIKE $1
        OR e.title ILIKE $1
        OR e.annotation ILIKE $1
        OR m.place ILIKE $1
        OR m.annotation ILIKE $1
      )

    UNION

    /* search editors */
    SELECT c.id, c.title
    FROM courses c JOIN editors e ON c.id = e.course_id
      JOIN users u ON u.id = e.user_id
    WHERE
      c.active
      AND c.expiration_date > now()
      AND (
        u.first_name ILIKE $1
        OR u.last_name ILIKE $1
        OR u.email ILIKE $1
      )

    UNION

    /* search instructors */
    SELECT c.id, c.title
    FROM courses c JOIN instructors i ON c.id = i.course_id
      JOIN users u ON u.id = i.user_id
    WHERE
      c.active
      AND c.expiration_date > now()
      AND (
        u.first_name ILIKE $1
        OR u.last_name ILIKE $1
        OR u.email ILIKE $1
      )

    UNION

    /* search groups */
    SELECT c.id, c.title
    FROM courses c
    WHERE
      c.active
      AND c.expiration_date > now()
      AND c.parent_id IN (
        SELECT g.id
        FROM groups g
        WHERE g.name ILIKE $1
      )

    UNION

    /* search creators */
    SELECT c.id, c.title
    FROM courses c JOIN users u ON c.creator = u.id
    WHERE
      c.active
      AND c.expiration_date > now()
      AND (
        u.first_name ILIKE $1
        OR u.last_name ILIKE $1
        OR u.email ILIKE $1
      )

    UNION

    /* search calendar events */
    SELECT c.id, c.title
    FROM courses c LEFT OUTER JOIN calendar_events e ON c.id = e.course_id
    WHERE
      c.active
      AND c.expiration_date > now()
      AND (
        e.title ILIKE $1
        OR e.annotation ILIKE $1
      )

  `

	stmtAllCoursesAdmin = `
		SELECT c.id, c.title,
			TO_CHAR (c.creation_date AT TIME ZONE $1, 'YYYY-MM-DD HH24:MI') as creation_date
		FROM courses c
		WHERE $2 = 0
	`

	stmtSearchCoursesForDraft = `
    /* search course fields */
    SELECT c.id, c.title
    FROM courses c LEFT OUTER JOIN events e ON c.id = e.course_id
      LEFT OUTER JOIN meetings m ON e.id = m.event_id
    WHERE (
        c.title ILIKE $1
        OR c.description ILIKE $1
        OR c.speaker ILIKE $1
        OR c.subtitle ILIKE $1
        OR e.title ILIKE $1
        OR e.annotation ILIKE $1
        OR m.place ILIKE $1
        OR m.annotation ILIKE $1
      )
			AND (
					c.id IN (
						SELECT e.course_id AS id
						FROM editors e
						WHERE e.user_id = $2
					)
				OR
					c.creator = $2
			)

    UNION

    /* search editors */
    SELECT c.id, c.title
    FROM courses c JOIN editors e ON c.id = e.course_id
      JOIN users u ON u.id = e.user_id
    WHERE (
        u.first_name ILIKE $1
        OR u.last_name ILIKE $1
        OR u.email ILIKE $1
      )
			AND (
					c.id IN (
						SELECT e.course_id AS id
						FROM editors e
						WHERE e.user_id = $2
					)
				OR
					c.creator = $2
			)

    UNION

    /* search instructors */
    SELECT c.id, c.title
    FROM courses c JOIN instructors i ON c.id = i.course_id
      JOIN users u ON u.id = i.user_id
    WHERE (
        u.first_name ILIKE $1
        OR u.last_name ILIKE $1
        OR u.email ILIKE $1
      )
			AND (
					c.id IN (
						SELECT e.course_id AS id
						FROM editors e
						WHERE e.user_id = $2
					)
				OR
					c.creator = $2
			)

    UNION

    /* search groups */
    SELECT c.id, c.title
    FROM courses c
    WHERE c.parent_id IN (
        SELECT g.id
        FROM groups g
        WHERE g.name ILIKE $1
      )
			AND (
					c.id IN (
						SELECT e.course_id AS id
						FROM editors e
						WHERE e.user_id = $2
					)
				OR
					c.creator = $2
			)

    UNION

    /* search creators */
    SELECT c.id, c.title
    FROM courses c JOIN users u ON c.creator = u.id
    WHERE (
        u.first_name ILIKE $1
        OR u.last_name ILIKE $1
        OR u.email ILIKE $1
      )
			AND (
					c.id IN (
						SELECT e.course_id AS id
						FROM editors e
						WHERE e.user_id = $2
					)
				OR
					c.creator = $2
			)

    UNION

    /* search calendar events */
    SELECT c.id, c.title
    FROM courses c LEFT OUTER JOIN calendar_events e ON c.id = e.course_id
    WHERE (
        e.title ILIKE $1
        OR e.annotation ILIKE $1
      )
			AND (
					c.id IN (
						SELECT e.course_id AS id
						FROM editors e
						WHERE e.user_id = $2
					)
				OR
					c.creator = $2
			)

  `

	stmtSearchCoursesForDraftAdmin = `
    /* search course fields */
    SELECT c.id, c.title
    FROM courses c LEFT OUTER JOIN events e ON c.id = e.course_id
      LEFT OUTER JOIN meetings m ON e.id = m.event_id
    WHERE (
        c.title ILIKE $1
        OR c.description ILIKE $1
        OR c.speaker ILIKE $1
        OR c.subtitle ILIKE $1
        OR e.title ILIKE $1
        OR e.annotation ILIKE $1
        OR m.place ILIKE $1
        OR m.annotation ILIKE $1
      )

    UNION

    /* search editors */
    SELECT c.id, c.title
    FROM courses c JOIN editors e ON c.id = e.course_id
      JOIN users u ON u.id = e.user_id
    WHERE (
        u.first_name ILIKE $1
        OR u.last_name ILIKE $1
        OR u.email ILIKE $1
      )

    UNION

    /* search instructors */
    SELECT c.id, c.title
    FROM courses c JOIN instructors i ON c.id = i.course_id
      JOIN users u ON u.id = i.user_id
    WHERE (
        u.first_name ILIKE $1
        OR u.last_name ILIKE $1
        OR u.email ILIKE $1
      )

    UNION

    /* search groups */
    SELECT c.id, c.title
    FROM courses c
    WHERE c.parent_id IN (
        SELECT g.id
        FROM groups g
        WHERE g.name ILIKE $1
      )

    UNION

    /* search creators */
    SELECT c.id, c.title
    FROM courses c JOIN users u ON c.creator = u.id
    WHERE (
        u.first_name ILIKE $1
        OR u.last_name ILIKE $1
        OR u.email ILIKE $1
      )

    UNION

    /* search calendar events */
    SELECT c.id, c.title
    FROM courses c LEFT OUTER JOIN calendar_events e ON c.id = e.course_id
    WHERE (
        e.title ILIKE $1
        OR e.annotation ILIKE $1
      )
  `
)
