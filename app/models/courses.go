package models

import (
	"strings"
	"turm/app"
)

/*Courses holds different courses. */
type Courses []Course

/*Search all courses. */
func (courses *Courses) Search(value string) (err error) {

	//we need to divide the string into substrings so we don't have whitespaces
	strSlice := strings.Split(value, " ")
	searchVal := ""
	for _, str := range strSlice {
		searchVal += "%" + str + "%"
	}

	err = app.Db.Select(courses, stmtSearchCourses, searchVal)
	if err != nil {
		log.Error("failed to search courses", "value", value, "searchVal",
			searchVal, "error", err.Error())
	}

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
)
