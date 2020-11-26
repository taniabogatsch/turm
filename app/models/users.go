package models

import (
	"strconv"
	"strings"
	"turm/app"

	"github.com/jmoiron/sqlx"
)

/*Users holds specific users, such as only admins, creators, ... */
type Users []User

/*Get specific users. */
func (users *Users) Get(role Role) (err error) {

	switch role {
	case ADMIN:
		err = app.Db.Select(users, stmtSelectUsers, ADMIN, app.TimeZone)
		if err != nil {
			log.Error("failed to get users", "role", ADMIN, "error", err.Error())
		}
	case CREATOR:
		err = app.Db.Select(users, stmtSelectUsers, CREATOR, app.TimeZone)
		if err != nil {
			log.Error("failed to get users", "role", CREATOR, "error", err.Error())
		}
	default:
		log.Error("invalid role provided", "role", role)
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

	err = app.Db.Select(users, stmtSearchUsers, values, matrNr, app.TimeZone, *searchInactive)
	if err != nil {
		log.Error("failed to search users", "values", values,
			"matrNr", matrNr, "error", err.Error())
	}
	return
}

/*AutoEnrollFromWaitList is a function that is triggered in many different situations:
- A user unsubscribes from an event and now there is a free slot.
- When increasing the course capacity (TODO).
- When manually unsubscribing an user from an event (TODO).
*/
func (users *Users) AutoEnrollFromWaitList(tx *sqlx.Tx, eventID *int,
	status EnrollmentStatus) (err error) {

	err = tx.Select(users, stmtAutoEnrollFromWaitlist, *eventID, status)
	if err != nil {
		log.Error("failed to auto enroll users from wait list", "eventID", *eventID,
			"status", status, "error", err.Error())
		tx.Rollback()
		return
	}

	//get data of each user (used for sending e-mails to auto enrolled users)
	for key := range *users {
		if err = (*users)[key].Get(tx); err != nil {
			return
		}
	}

	return
}

const (
	stmtSearchUsers = `
    SELECT id, last_name, first_name, email, salutation, role, title, academic_title, name_affix,
      TO_CHAR (last_login AT TIME ZONE $3, 'YYYY-MM-DD HH24:MI') as last_login
    FROM users
    WHERE (
				(activation_code IS NOT NULL) = $4
				OR activation_code IS NULL
			)
      AND (
        /* all combinations having a name_affix */
        title || academic_title || first_name || name_affix || last_name ILIKE $1
        OR title || first_name || name_affix || last_name ILIKE $1
        OR academic_title || first_name || name_affix || last_name ILIKE $1
        OR first_name || name_affix || last_name ILIKE $1

        /* all combinations without a name_affix */
        OR title || academic_title || first_name || last_name ILIKE $1
        OR title || first_name || last_name ILIKE $1
        OR academic_title || first_name || last_name ILIKE $1
        OR first_name || last_name ILIKE $1

        /* others */
        OR email ILIKE $1
        OR matr_nr = $2
      )
    ORDER BY last_name, first_name, id
    LIMIT 5
  `

	stmtSelectUsers = `
		SELECT id, last_name, first_name, email, salutation, title, academic_title, name_affix,
			TO_CHAR (last_login AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') as last_login
		FROM users
		WHERE role = $1
			AND activation_code IS NULL
		ORDER BY last_name, first_name, id
	`

	stmtAutoEnrollFromWaitlist = `
		UPDATE enrolled
		SET status = $2
		WHERE event_id = $1
			AND user_id IN (

				/* select all to-be-enrolled users */
				SELECT en.user_id
				FROM enrolled en
				WHERE en.event_id = $1
					AND en.status = 1 /* on waitlist */
				ORDER BY en.time_of_enrollment ASC
				LIMIT (

					/* get the number of free slots in the event */
					SELECT (ev.capacity - (

							SELECT
								CASE
									WHEN COUNT(e.user_id) <= ev.capacity
									THEN COUNT(e.user_id)
					        ELSE ev.capacity
					      END
							FROM enrolled e
							WHERE e.event_id = $1
								AND e.status != 1 /*on waitlist*/

						))
					FROM events ev
					WHERE ev.id = $1
				)
			)
		RETURNING user_id AS id
	`
)
