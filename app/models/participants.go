package models

import (
	"database/sql"
	"turm/app"

	"github.com/jmoiron/sqlx"
)

/*Participants of a course. */
type Participants struct {
	ID              int            `db:"id, primarykey, autoincrement"`
	Title           string         `db:"title"`
	Active          bool           `db:"active"`
	EnrollmentStart string         `db:"enrollment_start"`
	EnrollmentEnd   string         `db:"enrollment_end"`
	UnsubscribeEnd  sql.NullString `db:"unsubscribe_end"`
	ExpirationDate  string         `db:"expiration_date"`

	Expired bool
	Lists   ParticipantLists
}

/*Get all participants of a course. */
func (parts *Participants) Get() (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	//get course data
	err = tx.Get(parts, stmtSelectParticipantsCourseData, parts.ID, app.TimeZone)
	if err != nil {
		log.Error("failed to get participants course data", "parts ID", parts.ID,
			"error", err.Error())
		tx.Rollback()
		return
	}

	//get all participant lists of this course
	if err = parts.Lists.Get(tx, &parts.ID); err != nil {
		return
	}

	tx.Commit()
	return
}

/*ParticipantLists of a course. */
type ParticipantLists []ParticipantList

/*ParticipantList of an event. */
type ParticipantList struct {
	ID          int            `db:"id, primarykey, autoincrement"`
	CourseID    int            `db:"course_id"`
	Capacity    int            `db:"capacity"`
	HasWaitlist bool           `db:"has_waitlist"`
	Title       string         `db:"title"`
	Annotation  sql.NullString `db:"annotation"`

	//Fullness is the number of users that enrolled in this event
	Fullness int ``
	//Percentage is (Fullness * 100) / Capacity
	Percentage int ``

	//all participant lists of the event
	Participants Entries
	Waitlist     Entries
	Unsubscribed Entries
}

/*Get all participant lists of a course. */
func (lists *ParticipantLists) Get(tx *sqlx.Tx, partsID *int) (err error) {

	//get event data for lists
	err = tx.Select(lists, stmtSelectEventListsData, *partsID)
	if err != nil {
		log.Error("failed to get event lists data", "partsID", *partsID,
			"error", err.Error())
		tx.Rollback()
		return
	}

	//get the lists for each event
	for key, list := range *lists {

		//set the percentage field
		(*lists)[key].Percentage = (list.Fullness * 100) / list.Capacity

		//participants list
		if err = (*lists)[key].Participants.Get(tx, "participants", &list.ID); err != nil {
			return
		}

		//wait list (if exists)
		if (*lists)[key].HasWaitlist {
			if err = (*lists)[key].Waitlist.Get(tx, "waitlist", &list.ID); err != nil {
				return
			}
		}

		//get unsubscribed list
		if err = (*lists)[key].Unsubscribed.Get(tx, "unsubscribed", &list.ID); err != nil {
			return
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
func (entries *Entries) Get(tx *sqlx.Tx, listType string, eventID *int) (err error) {

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
	}

	return
}

const (
	stmtSelectParticipantsCourseData = `
    SELECT
      id, title, active,
      TO_CHAR (enrollment_start AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') AS enrollment_start,
      TO_CHAR (enrollment_end AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') AS enrollment_end,
      TO_CHAR (unsubscribe_end AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') AS unsubscribe_end,
      TO_CHAR (expiration_date AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') AS expiration_date,
      (current_timestamp >= expiration_date) AS expired
    FROM courses
    WHERE id = $1
  `

	stmtSelectEventListsData = `
    SELECT
      e.id, e.course_id, e.capacity, e.has_waitlist,
      e.title,
      (
        SELECT COUNT(en.user_id)
        FROM enrolled en
        WHERE en.event_id = e.id
          AND status != 1 /*on waitlist*/
      ) AS fullness
    FROM events e
    WHERE e.course_id = $1
  `

	stmtSelectParticipants = `
    SELECT
      u.id, u.last_name, u.first_name, u.email, u.salutation,
      u.language, u.matr_nr, u.academic_title, u.title, u.name_affix, u.affiliations,
      e.user_id, e.event_id, e.status, e.email_traffic,
      TO_CHAR (e.time_of_enrollment AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') AS time_of_enrollment
    FROM users u JOIN enrolled e ON u.id = e.user_id
    WHERE e.event_id = $1
      AND e.status != 1 /*on waitlist */
  `

	stmtSelectParticipantsWaitlist = `
    SELECT
      u.id, u.last_name, u.first_name, u.email, u.salutation,
      u.language, u.matr_nr, u.academic_title, u.title, u.name_affix, u.affiliations,
      e.user_id, e.event_id, e.status, e.email_traffic,
      TO_CHAR (e.time_of_enrollment AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') AS time_of_enrollment
    FROM users u JOIN enrolled e ON u.id = e.user_id
    WHERE e.event_id = $1
      AND e.status = 1 /*on waitlist */
  `

	stmtSelectUnsubscribed = `
    SELECT
      u.id, u.last_name, u.first_name, u.email, u.salutation,
      u.language, u.matr_nr, u.academic_title, u.title, u.name_affix, u.affiliations
    FROM users u JOIN unsubscribed un ON u.id = un.user_id
    WHERE un.event_id = $1
  `
)
