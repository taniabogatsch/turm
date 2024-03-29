package models

import (
	"database/sql"
	"strconv"
	"strings"
	"time"
	"turm/app"

	"github.com/jmoiron/sqlx"
)

/*ListConf determines to whom an e-mail is send or which users are at the
downloaded participants list. */
type ListConf struct {

	//specify which users are downloaded/e-mailed
	AllEvents    bool
	EventIDs     []int
	Participants bool
	WaitList     bool
	Unsubscribed bool

	//used for downloading the participants list
	UseComma bool
	Filename string

	//used for sending an e-mail
	Subject string
	Content string

	//used to specify a time interval for calendar events
	Start     string
	End       string
	StartTime string
	EndTime   string
}

/*Participants of a course. */
type Participants struct {
	ID                 int            `db:"id, primarykey, autoincrement"`
	Title              string         `db:"title"`
	Active             bool           `db:"active"`
	EnrollmentStartStr string         `db:"enrollment_start_str"`
	EnrollmentEndStr   string         `db:"enrollment_end_str"`
	UnsubscribeEndStr  sql.NullString `db:"unsubscribe_end_str"`
	ExpirationDateStr  string         `db:"expiration_date_str"`
	ViewMatrNr         bool           `db:"view_matr_nr"`
	UserEMail          string         `db:"user_email"`

	Expired bool
	Lists   ParticipantLists
}

/*Get all participants of a course. */
func (parts *Participants) Get(userID int, allSlots bool) (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	//get course data
	err = tx.Get(parts, stmtSelectParticipantsCourseData, parts.ID,
		app.TimeZone, userID)
	if err != nil {
		log.Error("failed to get participants course data", "parts ID", parts.ID,
			"userID", userID, "error", err.Error())
		tx.Rollback()
		return
	}

	//get whether the user is allowed to see matriculation numbers
	err = tx.Get(parts, stmtGetViewMatrNr, parts.ID, userID)
	if err != nil {
		log.Error("failed to get whether user is allowed to see matr nr or not",
			"partsID", parts.ID, "userID", userID, "error", err.Error())
		tx.Rollback()
		return
	}

	//get all participant lists of this course
	if err = parts.Lists.Get(tx, &parts.ID, parts.ViewMatrNr, allSlots); err != nil {
		return
	}

	tx.Commit()
	return
}

/*ParticipantLists of a course. */
type ParticipantLists []ParticipantList

/*ParticipantList of an event. */
type ParticipantList struct {

	//general event information
	Event

	//all participant lists of the event
	Participants Entries
	Waitlist     Entries
	Unsubscribed Entries

	//additional calendar event information
	IsCalendarEvent bool `db:"is_calendar_event"`
	Monday          time.Time
	Week            int
	Year            int
	//day templates for this week [0...6]
	Days Days

	//all slots of a calendar event
	Slots Slots
}

/*Get all participant lists of a course. */
func (lists *ParticipantLists) Get(tx *sqlx.Tx, courseID *int, viewMatrNr bool,
	allSlots bool) (err error) {

	//get event data for lists
	err = tx.Select(lists, stmtSelectEventData, *courseID)
	if err != nil {
		log.Error("failed to get event lists data", "courseID", *courseID,
			"error", err.Error())
		tx.Rollback()
		return
	}

	//get the lists for each event
	for key, list := range *lists {

		//get all slots or only those of a specific day
		if list.IsCalendarEvent {

			//TODO: make this more efficient: only get slots in interval from DB

			if allSlots {

				//get the slots
				err = (*lists)[key].Slots.GetAllCalendarEvent(tx, list.ID)
				if err != nil {
					return
				}

				for idx := range (*lists)[key].Slots {

					//get user data
					(*lists)[key].Slots[idx].User.ID = (*lists)[key].Slots[idx].UserID
					if err = (*lists)[key].Slots[idx].User.Get(tx); err != nil {
						return
					}

					//create dummy matriculation numbers, if the user is not allowed to see them
					if (*lists)[key].Slots[idx].User.MatrNr.Valid && !viewMatrNr {
						(*lists)[key].Slots[idx].User.MatrNr.Int32 = 12345
					}
				}

			} else {

				//get the last (current) monday
				now := time.Now()
				weekday := time.Now().Weekday()
				monday := now.AddDate(0, 0, -1*(int(weekday)-1))

				//set the current week
				(*lists)[key].Monday = monday
				_, (*lists)[key].Week = monday.ISOWeek()
				(*lists)[key].Year = monday.Year()

				//get the slots of each day
				err = (*lists)[key].Days.Get(tx, &list.ID, monday, true, viewMatrNr, 0)
				if err != nil {
					return
				}
			}

		} else { //get all user lists for normal events

			//set the percentage field
			(*lists)[key].Percentage = (list.Fullness * 100) / list.Capacity

			//participants list
			err = (*lists)[key].Participants.Get(tx, "participants", &list.ID, viewMatrNr)
			if err != nil {
				return
			}

			//wait list (if exists)
			if (*lists)[key].HasWaitlist {
				err = (*lists)[key].Waitlist.Get(tx, "waitlist", &list.ID, viewMatrNr)
				if err != nil {
					return
				}
			}

			//get unsubscribed list
			err = (*lists)[key].Unsubscribed.Get(tx, "unsubscribed", &list.ID, viewMatrNr)
			if err != nil {
				return
			}
		}

	}
	return
}

/*Entries of all users on either the participants list, the wait list or
the unsubscribed list. */
type Entries []Entry

/*Entry of a user on either the participants list, the wait list or
the unsubscribed list. */
type Entry struct {
	User
	Enrolled
}

/*Get all entries on a specific list. */
func (entries *Entries) Get(tx *sqlx.Tx, listType string, eventID *int, viewMatrNr bool) (err error) {

	switch listType {
	case "participants":
		err = tx.Select(entries, stmtSelectParticipants, *eventID, app.TimeZone)
	case "waitlist":
		err = tx.Select(entries, stmtSelectParticipantsWaitlist, *eventID, app.TimeZone)
	case "unsubscribed":
		err = tx.Select(entries, stmtSelectUnsubscribed, *eventID)
	}

	if err != nil {
		log.Error("failed to get entries", "listType", listType, "eventID", *eventID,
			"error", err.Error())
		tx.Rollback()
		return
	}

	for key := range *entries {

		//get the courses of studies
		if (*entries)[key].IsLDAP {
			err = (*entries)[key].Studies.Select(tx, &(*entries)[key].ID)
			if err != nil {
				return
			}
		}

		//create dummy matriculation numbers, if the user is not allowed to see them
		if (*entries)[key].MatrNr.Valid && !viewMatrNr {
			(*entries)[key].MatrNr.Int32 = 12345
		}
	}

	return
}

/*Search all entries. */
func (entries *Entries) Search(ID, eventID, userID *int, value *string) (hasWaitlist bool, err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	//get whether the user is allowed to see matriculation numbers
	var viewMatrNr bool
	err = tx.Get(&viewMatrNr, stmtGetViewMatrNr, *ID, userID)
	if err != nil {
		log.Error("failed to get whether user is allowed to see matr nr or not",
			"ID", *ID, "userID", *userID, "error", err.Error())
		tx.Rollback()
		return
	}

	//prepare the search value for pattern matching
	values := "%"
	for _, val := range strings.Split(*value, " ") {
		values += val + "%"
	}
	//the value can be the matriculation number
	matrNr, _ := strconv.Atoi(*value) //matrNr is 0 if there is an error

	err = tx.Select(entries, stmtSearchEntries, values, matrNr, *eventID)
	if err != nil {
		log.Error("failed to search entries", "values", values,
			"matrNr", matrNr, "eventID", *eventID, "error", err.Error())
		tx.Rollback()
		return
	}

	//create dummy matriculation numbers, if the user is not allowed to see them
	if !viewMatrNr {
		for key := range *entries {
			if (*entries)[key].MatrNr.Valid {
				(*entries)[key].MatrNr.Int32 = 12345
			}
		}
	}

	event := Event{ID: *eventID}
	if err = event.GetColumnValue(tx, "has_waitlist"); err != nil {
		return
	}
	hasWaitlist = event.HasWaitlist

	tx.Commit()
	return
}

const (
	stmtSelectParticipantsCourseData = `
    SELECT
      id, title, active,
      TO_CHAR (enrollment_start AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') AS enrollment_start_str,
      TO_CHAR (enrollment_end AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') AS enrollment_end_str,
      TO_CHAR (expiration_date AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') AS expiration_date_str,
      (current_timestamp >= expiration_date) AS expired,

			( SELECT email
				FROM users
				WHERE id = $3
			) AS user_email,

			CASE WHEN unsubscribe_end IS NOT NULL
					THEN TO_CHAR (unsubscribe_end AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI')
				ELSE null
			END AS unsubscribe_end_str

    FROM courses
    WHERE id = $1
  `

	stmtSelectEventData = `
    (SELECT
      e.id, e.course_id, e.capacity, e.has_waitlist, e.has_comments,
      e.title, e.annotation, false AS is_calendar_event,
      (
        SELECT COUNT(en.user_id)
        FROM enrolled en
        WHERE en.event_id = e.id
          AND status != 1 /*on waitlist*/
      ) AS fullness
    FROM events e
    WHERE e.course_id = $1)

		UNION ALL

		(SELECT
			e.id, e.course_id, 0 AS capacity, false AS has_waitlist, false AS has_comments,
			e.title, e.annotation, true AS is_calendar_event,
			0 AS fullness
		FROM calendar_events e
		WHERE e.course_id = $1)

		ORDER BY id ASC
  `

	stmtSelectParticipants = `
    SELECT
      u.id, u.last_name, u.first_name, u.email, u.salutation, (u.password IS NULL) AS is_ldap,
      u.language, u.matr_nr, u.academic_title, u.title, u.name_affix, u.affiliations,
      e.user_id, e.event_id, e.status, e.time_of_enrollment, e.comment,
      TO_CHAR (e.time_of_enrollment AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') AS time_of_enrollment_str
    FROM users u JOIN enrolled e ON u.id = e.user_id
    WHERE e.event_id = $1
      AND e.status != 1 /*on waitlist */
		ORDER BY u.last_name ASC
  `

	stmtSelectParticipantsWaitlist = `
    SELECT
      u.id, u.last_name, u.first_name, u.email, u.salutation, (u.password IS NULL) AS is_ldap,
      u.language, u.matr_nr, u.academic_title, u.title, u.name_affix, u.affiliations,
      e.user_id, e.event_id, e.status, e.time_of_enrollment, e.comment,
      TO_CHAR (e.time_of_enrollment AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') AS time_of_enrollment_str
    FROM users u JOIN enrolled e ON u.id = e.user_id
    WHERE e.event_id = $1
      AND e.status = 1 /*on waitlist */
		ORDER BY u.last_name ASC
  `

	stmtSelectUnsubscribed = `
    SELECT
      u.id, u.last_name, u.first_name, u.email, u.salutation, (u.password IS NULL) AS is_ldap,
      u.language, u.matr_nr, u.academic_title, u.title, u.name_affix, u.affiliations,
			un.event_id, 5 AS status
    FROM users u JOIN unsubscribed un ON u.id = un.user_id
    WHERE un.event_id = $1
		ORDER BY u.last_name ASC
  `

	stmtGetViewMatrNr = `
		SELECT EXISTS (
			(
				SELECT id
				FROM courses
				WHERE id = $1
					AND creator = $2
			)

			UNION ALL

			(
				SELECT user_id AS id
				FROM editors
				WHERE course_id = $1
					AND user_id = $2
					AND view_matr_nr
			)

			UNION ALL

			(
				SELECT user_id AS id
				FROM instructors
				WHERE course_id = $1
					AND user_id = $2
					AND view_matr_nr
			)

		) AS view_matr_nr
	`

	stmtGetViewMatrNrCalendarEvent = `
		SELECT EXISTS (
			(
				SELECT id
				FROM courses
				WHERE id = (
						SELECT course_id
						FROM calendar_events
						WHERE id = $1
					)
					AND creator = $2
			)

			UNION ALL

			(
				SELECT user_id AS id
				FROM editors
				WHERE course_id = (
						SELECT course_id
						FROM calendar_events
						WHERE id = $1
					)
					AND user_id = $2
					AND view_matr_nr
			)

			UNION ALL

			(
				SELECT user_id AS id
				FROM instructors
				WHERE course_id = (
						SELECT course_id
						FROM calendar_events
						WHERE id = $1
					)
					AND user_id = $2
					AND view_matr_nr
			)

		) AS view_matr_nr
	`

	stmtSearchEntries = `
		SELECT u.id, u.last_name, u.first_name, u.email, u.salutation, u.title,
			u.academic_title, u.name_affix, u.matr_nr, u.affiliations, (u.password IS NULL) AS is_ldap,

			CASE WHEN en.status IS NOT NULL THEN en.status
            ELSE 5
      END AS status

		FROM users u LEFT OUTER JOIN enrolled en ON (u.id = en.user_id AND en.event_id = $3)
		WHERE u.activation_code IS NULL
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
)
