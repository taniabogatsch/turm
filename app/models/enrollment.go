package models

import (
	"turm/app"

	"github.com/jmoiron/sqlx"
)

/*EnrollmentStatus is a type for encoding the enrollment status. */
type EnrollmentStatus int

const (
	//ENROLLED users enrolled in an event
	ENROLLED EnrollmentStatus = iota
	//ONWAITLIST users are at the waitlist of an event
	ONWAITLIST
	//AWAITINGPAYMENT users enrolled in an event but did not yet pay the fee of the course
	AWAITINGPAYMENT
	//PAID users enrolled in an event and did pay the fee of the course
	PAID
	//FREED users enrolled in an event and do not have to pay the fee of the course
	FREED
	//UNSUBSCRIBED users unsubscribed from an event
	UNSUBSCRIBED
)

func (status EnrollmentStatus) String() string {
	return [...]string{"enrolled", "on waitlist", "awaiting payment",
		"paid", "freed", "unsubscribed"}[status]
}

/*EnrollOption is a type for encoding different enrollment options. */
type EnrollOption int

const (
	//ENROLL is for normally enrolling in an event
	ENROLL EnrollOption = iota
	//UNSUBSCRIBE is for normally unsubscribing from an event
	UNSUBSCRIBE
	//NOENROLL disables the enrollment button
	NOENROLL
	//NOUNSUBSCRIBE disables the unsubscribe button
	NOUNSUBSCRIBE
	//ENROLLTOWAITLIST is for enrolling to the wait list
	ENROLLTOWAITLIST
	//UNSUBSCRIBEFROMWAITLIST is for unsubscribing from the wait list
	UNSUBSCRIBEFROMWAITLIST
)

func (s EnrollOption) String() string {
	return [...]string{"enroll", "unsubscribe", "noenroll", "nounsubscribe", "enrolltowaitlist"}[s]
}

/*Enrollments of a user. */
type Enrollments []Enrolled

/*Enrolled is a model of the enrolled table. */
type Enrolled struct {
	UserID           int              `db:"user_id, primarykey"`
	EventID          int              `db:"event_id, primarykey"`
	Status           EnrollmentStatus `db:"status"`
	EMailTraffic     bool             `db:"email_traffic"`
	TimeOfEnrollment string           `db:"time_of_enrollment"`
}

/*GetByCourse all enrollments of a user for a specific course. */
func (enrollments *Enrollments) GetByCourse(tx *sqlx.Tx, userID, courseID *int) (err error) {

	err = tx.Select(enrollments, stmtSelectCourseEnrollments,
		*userID, *courseID)
	if err != nil {
		log.Error("failed to get enrollments of user", "userID", *userID,
			"courseID", *courseID, "error", err.Error())
		tx.Rollback()
	}
	return
}

/*Unsubscribed is a model of the unsubscribed table. */
type Unsubscribed struct {
	UserID  int `db:"user_id, primarykey"`
	EventID int `db:"event_id, primarykey"`
}

/*CourseStatus validates the enrollment status of an user for a course. */
type CourseStatus struct {
	AtBlacklist             bool
	AtWhitelist             bool
	UnsubscribeOver         bool `db:"unsubscribe_over"`
	NoEnrollmentPeriod      bool `db:"no_enrollment_period"`
	NotSatisfyRestrictions  bool
	NotLDAP                 bool
	MaxEnrollCoursesReached bool
	MaxEnrollCourses        int `db:"limit"`
}

/*EventStatus validates the enrollment status of an user for a event. */
type EventStatus struct {
	Enrolled           bool
	Full               bool
	EnrollLimitReached bool //important to evaluate EnrollLimitEvents
	OnWaitlist         bool
	InOtherEvent       bool
}

/*EnrollOrUnsubscribe a user in/from an event. */
func EnrollOrUnsubscribe(userID, eventID *int, action EnrollOption,
	key string) (data EMailData, waitList bool, users Users, msg string, err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	//get relevant event information
	event := Event{ID: *eventID}
	if err = event.Get(tx); err != nil {
		return
	}

	//get relevant course information
	course := Course{ID: event.CourseID}
	if err = course.GetForEnrollment(tx, userID, eventID); err != nil {
		return
	}

	//get the event enroll status
	if err = event.validateEnrollStatus(tx, userID, &course.EnrollLimitEvents); err != nil {
		return
	}

	//validate if allowed to enroll (to wait list)
	event.validateEnrollment(&course)

	if event.EnrollOption == NOENROLL || event.EnrollOption == NOUNSUBSCRIBE {
		//the user is not allowed to enroll or unsubscribe
		msg = event.EnrollMsg
		tx.Rollback()
		return
	}

	//enroll the user
	if action == ENROLL {

		if event.EnrollOption == UNSUBSCRIBE || event.EnrollOption == UNSUBSCRIBEFROMWAITLIST {
			//the user is already enrolled in this event
			msg = "validation.enrollment.already.enrolled"
			tx.Rollback()
			return
		}

		//set enroll status
		status := ENROLLED
		if course.Fee.Valid {
			status = AWAITINGPAYMENT
		}
		if event.EnrollOption == ENROLLTOWAITLIST {
			status = ONWAITLIST
			waitList = true
		}

		//validate enrollment key (if required)
		if event.EnrollmentKey.Valid {
			var validKey bool
			if err = tx.Get(&validKey, stmtValidateEnrollmentKey, *eventID, key); err != nil {
				log.Error("failed to validate enrollment key", "eventID", *eventID, "key", key,
					"error", err.Error())
				tx.Rollback()
				return
			} else if !validKey {
				msg = "validation.enrollment.invalid.key"
				tx.Rollback()
				return
			}
		}

		//enroll
		if _, err = tx.Exec(stmtEnrollUser, *userID, *eventID, status); err != nil {
			log.Error("failed to enroll user", "userID", *userID, "eventID", *eventID,
				"status", status, "error", err.Error())
			tx.Rollback()
			return
		}

		//try to remove the user from the unsubscribed table
		if _, err = tx.Exec(stmtDeleteUserFromUnsubscribed, *userID, *eventID); err != nil {
			log.Error("failed to enroll user", "userID", *userID, "eventID", *eventID,
				"error", err.Error())
			tx.Rollback()
			return
		}

	} else { //unsubscribe the user

		if event.EnrollOption == ENROLL || event.EnrollOption == ENROLLTOWAITLIST {
			//the user is already unsubscribed from this event
			msg = "validation.enrollment.already.unsubscribed"
			tx.Rollback()
			return
		}

		if event.EnrollOption == UNSUBSCRIBEFROMWAITLIST {
			waitList = true
		}

		//unsubscribe
		if _, err = tx.Exec(stmtUnsubscribeUser, *userID, *eventID); err != nil {
			log.Error("failed to unsubscribe user", "userID", *userID, "eventID", *eventID,
				"error", err.Error())
			tx.Rollback()
			return
		}

		//insert into unsubscribed table
		if _, err = tx.Exec(stmtInsertUserIntoUnsubscribed, *userID, *eventID); err != nil {
			log.Error("failed to enroll user", "userID", *userID, "eventID", *eventID,
				"error", err.Error())
			tx.Rollback()
			return
		}

		//handle users who get enrolled from the wait list
		if event.HasWaitlist {

			status := ENROLLED
			if course.Fee.Valid {
				status = AWAITINGPAYMENT
			}

			if err = users.AutoEnrollFromWaitList(tx, eventID, status); err != nil {
				return
			}
		}
	}

	//set e-mail data
	data.CourseTitle = course.Title
	data.EventTitle = event.Title
	data.CourseID = course.ID
	data.User.ID = *userID
	if err = data.User.Get(tx); err != nil {
		return
	}

	tx.Commit()
	return
}

const (
	stmtSelectCourseEnrollments = `
		SELECT en.user_id, en.event_id, en.status
		FROM enrolled en JOIN
			events e ON en.event_id = e.id
		WHERE en.user_id = $1
			AND e.course_id = $2
	`

	stmtGetCountUserEnrollments = `

		/* get all children */
		WITH RECURSIVE path (id, parent_id, course_limit)
			AS (
				/* starting entry */
				SELECT id, parent_id, course_limit
				FROM groups
				WHERE id = (

					/* get the starting group id */
					WITH RECURSIVE path (parent_id, id, course_limit)
						AS (
							/* starting entry */
							SELECT parent_id, id, course_limit
							FROM groups
							WHERE id = $1
								AND course_limit IS NULL

							UNION ALL

							/* construct path */
							SELECT g.parent_id, g.id, g.course_limit
							FROM groups g, path p
							WHERE p.parent_id = g.id
								AND g.course_limit IS NULL
						)

					/* select the root element of the constructed path */
					(SELECT parent_id AS id FROM path
					UNION ALL
					SELECT id FROM groups WHERE id = $1 AND course_limit IS NOT NULL)
					ORDER BY id DESC LIMIT 1
				)

				UNION ALL

				/* collect all children */
				SELECT g.id, g.parent_id, g.course_limit
				FROM groups g, path p
				WHERE p.id = g.parent_id
			)

		/* count the number of enrollments of the user in any of the children */
		SELECT COUNT(DISTINCT ev.course_id) AS enrollments
		FROM enrolled e JOIN
			events ev ON e.event_id = ev.id JOIN
			courses c ON ev.course_id = c.id JOIN
			path p ON c.parent_id = p.id
		WHERE e.user_id = $2
	`

	stmtEnrollUser = `
		INSERT INTO enrolled
			(user_id, event_id, status)
		VALUES ($1, $2, $3)
	`

	stmtDeleteUserFromUnsubscribed = `
		DELETE FROM unsubscribed
		WHERE user_id = $1
			AND event_id = $2
	`

	stmtUnsubscribeUser = `
		DELETE FROM enrolled
		WHERE user_id = $1
			AND event_id = $2
	`

	stmtInsertUserIntoUnsubscribed = `
		INSERT INTO unsubscribed
			(user_id, event_id)
		VALUES ($1, $2)
	`

	stmtValidateEnrollmentKey = `
		SELECT EXISTS (
			SELECT true
			FROM events
			WHERE id = $1
				AND enrollment_key = CRYPT($2, enrollment_key)
		) AS valid_key
	`
)
