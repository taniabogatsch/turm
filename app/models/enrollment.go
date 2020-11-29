package models

import (
	"turm/app"

	"github.com/jmoiron/sqlx"
	"github.com/revel/revel"
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
	TimeOfEnrollment string           `db:"time_of_enrollment"`

	//used for profile page
	CourseID    int    `db:"course_id"`
	CourseTitle string `db:"course_title"`
	EventTitle  string `db:"event_title"`
}

/*SelectByCourse selects all enrollments of a user for a specific course. */
func (enrollments *Enrollments) SelectByCourse(tx *sqlx.Tx, userID, courseID *int) (err error) {

	err = tx.Select(enrollments, stmtSelectCourseEnrollments,
		*userID, *courseID)
	if err != nil {
		log.Error("failed to get enrollments of user", "userID", *userID,
			"courseID", *courseID, "error", err.Error())
		tx.Rollback()
	}
	return
}

/*SelectByUser returns all enrollments of a user. */
func (enrollments *Enrollments) SelectByUser(tx *sqlx.Tx, userID *int, expired bool) (err error) {

	if expired {
		err = tx.Select(enrollments, stmtSelectUserEnrollmentsExpired, *userID,
			app.TimeZone)
	} else {
		err = tx.Select(enrollments, stmtSelectUserEnrollments, *userID,
			app.TimeZone)
	}

	if err != nil {
		log.Error("failed to select user enrollments", "userID", *userID,
			"error", err.Error())
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
func (enrolled *Enrolled) EnrollOrUnsubscribe(action EnrollOption, key string) (data EMailData,
	waitList bool, users Users, msg string, err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	//get relevant event information
	event := Event{ID: enrolled.EventID}
	if err = event.Get(tx); err != nil {
		return
	}

	//get relevant course information
	course := Course{ID: event.CourseID}
	err = course.GetForEnrollment(tx, &enrolled.UserID, &enrolled.EventID)
	if err != nil {
		return
	}

	//get the event enroll status
	err = event.validateEnrollStatus(tx, &enrolled.UserID, &course.EnrollLimitEvents)
	if err != nil {
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
		enrolled.Status = ENROLLED
		if course.Fee.Valid {
			enrolled.Status = AWAITINGPAYMENT
		}
		if event.EnrollOption == ENROLLTOWAITLIST {
			enrolled.Status = ONWAITLIST
			waitList = true
		}

		//validate enrollment key (if required)
		if event.EnrollmentKey.Valid {
			var validKey bool
			err = tx.Get(&validKey, stmtValidateEnrollmentKey, enrolled.EventID, key)
			if err != nil {
				log.Error("failed to validate enrollment key", "eventID", enrolled.EventID,
					"key", key, "error", err.Error())
				tx.Rollback()
				return
			} else if !validKey {
				msg = "validation.enrollment.invalid.key"
				tx.Rollback()
				return
			}
		}

		//enroll
		if err = enrolled.enroll(tx); err != nil {
			return
		}
		if err = enrolled.removeFromUnsubscribed(tx); err != nil {
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

		if err = enrolled.unsubscribe(tx); err != nil {
			return
		}

		//handle users who get enrolled from the wait list
		if event.HasWaitlist {

			status := ENROLLED
			if course.Fee.Valid {
				status = AWAITINGPAYMENT
			}

			err = users.AutoEnrollFromWaitList(tx, &enrolled.EventID, status)
			if err != nil {
				return
			}
		}
	}

	//set e-mail data
	data.CourseTitle = course.Title
	data.EventTitle = event.Title
	data.CourseID = course.ID
	data.User.ID = enrolled.UserID
	if err = data.User.Get(tx); err != nil {
		return
	}

	tx.Commit()
	return
}

/*Enroll a user via participants management. */
func (enrolled *Enrolled) Enroll(courseID *int, v *revel.Validation) (data EMailData, err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	//get if already enrolled
	enrolled.Status = ENROLLED
	if isEnrolled, err := enrolled.hasEventStatus(tx); err != nil {
		return data, err
	} else if isEnrolled {
		v.ErrorKey("validation.enrollment.manual.already.enrolled")
		tx.Rollback()
		return data, err
	}

	course := Course{ID: *courseID}
	if err = course.GetColumnValue(tx, "title"); err != nil {
		return
	}
	if err = course.GetColumnValue(tx, "fee"); err != nil {
		return
	}
	event := Event{ID: enrolled.EventID}
	if err = event.GetColumnValue(tx, "title"); err != nil {
		return
	}

	//get if on wait list
	enrolled.Status = ONWAITLIST
	waitlist, err := enrolled.hasEventStatus(tx)
	if err != nil {
		return
	}

	//set enroll status
	enrolled.Status = ENROLLED
	if course.Fee.Valid {
		enrolled.Status = AWAITINGPAYMENT
	}

	if waitlist {

		//update status
		if err = enrolled.updateStatus(tx); err != nil {
			return
		}

	} else {

		//enroll
		if err = enrolled.enroll(tx); err != nil {
			return
		}
		if err = enrolled.removeFromUnsubscribed(tx); err != nil {
			return
		}
	}

	//set e-mail data
	data.CourseTitle = course.Title
	data.EventTitle = event.Title
	data.CourseID = course.ID
	data.User.ID = enrolled.UserID
	if err = data.User.Get(tx); err != nil {
		return
	}

	tx.Commit()
	return
}

/*Waitlist enrolls a user to a wait list via participants management. */
func (enrolled *Enrolled) Waitlist(courseID *int, v *revel.Validation) (data EMailData,
	users Users, err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	course := Course{ID: *courseID}
	if err = course.GetColumnValue(tx, "title"); err != nil {
		return
	}
	if err = course.GetColumnValue(tx, "fee"); err != nil {
		return
	}

	//get relevant event information
	event := Event{ID: enrolled.EventID}
	if err = event.Get(tx); err != nil {
		return
	}

	if !event.HasWaitlist {
		v.ErrorKey("validation.enrollment.manual.no.wait.list")
		tx.Rollback()
		return
	}
	if event.Fullness < event.Capacity {
		v.ErrorKey("validation.enrollment.manual.wait.list.invalid")
		tx.Rollback()
		return
	}

	//get if on wait list
	enrolled.Status = ONWAITLIST
	if waitlist, err := enrolled.hasEventStatus(tx); err != nil {
		return data, users, err
	} else if waitlist {
		v.ErrorKey("validation.enrollment.manual.already.at.wait.list")
		tx.Rollback()
		return data, users, err
	}

	//get if already enrolled
	enrolled.Status = ENROLLED
	isEnrolled, err := enrolled.hasEventStatus(tx)
	if err != nil {
		return
	}

	if isEnrolled && event.Fullness == event.Capacity {

		//validate whether the user would be auto enrolled directly after
		//being put at the wait list

		autoEnroll := false
		err = tx.Get(&autoEnroll, stmtUserWillGetAutoEnrolled,
			enrolled.EventID, enrolled.UserID)
		if err != nil {
			log.Error("failed to get if user will get auto enrolled", "enrolled", *enrolled,
				"isEnrolled", isEnrolled, "error", err.Error())
			tx.Rollback()
			return
		} else if autoEnroll {
			v.ErrorKey("validation.enrollment.manual.auto.enrolled")
			tx.Rollback()
			return
		}
	}

	enrolled.Status = ONWAITLIST

	if isEnrolled {

		//update status
		if err = enrolled.updateStatus(tx); err != nil {
			return
		}

	} else {

		//enroll to wait list
		if err = enrolled.enroll(tx); err != nil {
			return
		}
		if err = enrolled.removeFromUnsubscribed(tx); err != nil {
			return
		}
	}

	//handle users who get enrolled from the wait list
	if event.HasWaitlist && isEnrolled {

		status := ENROLLED
		if course.Fee.Valid {
			status = AWAITINGPAYMENT
		}

		err = users.AutoEnrollFromWaitList(tx, &enrolled.EventID, status)
		if err != nil {
			return
		}
	}

	//set e-mail data
	data.CourseTitle = course.Title
	data.EventTitle = event.Title
	data.CourseID = course.ID
	data.User.ID = enrolled.UserID
	if err = data.User.Get(tx); err != nil {
		return
	}

	tx.Commit()
	return
}

/*Unsubscribe a user via participants management. */
func (enrolled *Enrolled) Unsubscribe(courseID *int, v *revel.Validation) (data EMailData,
	users Users, err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	course := Course{ID: *courseID}
	if err = course.GetColumnValue(tx, "title"); err != nil {
		return
	}
	if err = course.GetColumnValue(tx, "fee"); err != nil {
		return
	}

	//get relevant event information
	event := Event{ID: enrolled.EventID}
	if err = event.Get(tx); err != nil {
		return
	}

	//get if on wait list
	enrolled.Status = ONWAITLIST
	waitList, err := enrolled.hasEventStatus(tx)
	if err != nil {
		return
	}

	if !waitList {
		//get if already enrolled
		enrolled.Status = ENROLLED
		isEnrolled, err := enrolled.hasEventStatus(tx)
		if err != nil {
			return data, users, err
		} else if !isEnrolled {
			v.ErrorKey("validation.enrollment.manual.already.unsubscribed")
			tx.Rollback()
			return data, users, err
		}
	}

	if err = enrolled.unsubscribe(tx); err != nil {
		return
	}

	//handle users who get enrolled from the wait list
	if event.HasWaitlist && !waitList {

		status := ENROLLED
		if course.Fee.Valid {
			status = AWAITINGPAYMENT
		}

		err = users.AutoEnrollFromWaitList(tx, &enrolled.EventID, status)
		if err != nil {
			return
		}
	}

	//set e-mail data
	data.CourseTitle = course.Title
	data.EventTitle = event.Title
	data.CourseID = course.ID
	data.User.ID = enrolled.UserID
	if err = data.User.Get(tx); err != nil {
		return
	}

	tx.Commit()
	return
}

/*ChangeStatus updates the enrollment status of a user. */
func (enrolled *Enrolled) ChangeStatus(courseID *int, v *revel.Validation) (data EMailData,
	err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	course := Course{ID: *courseID}
	if err = course.GetColumnValue(tx, "title"); err != nil {
		return
	}
	if err = course.GetColumnValue(tx, "fee"); err != nil {
		return
	}

	if !course.Fee.Valid {
		v.ErrorKey("validation.enrollment.change.status.invalid")
		tx.Rollback()
		return
	}

	event := Event{ID: enrolled.EventID}
	if err = event.GetColumnValue(tx, "title"); err != nil {
		return
	}

	//change status
	err = tx.Get(enrolled, stmtUpdateStatus, enrolled.EventID, enrolled.UserID,
		enrolled.Status)
	if err != nil {
		log.Error("failed to update payment status", "enrolled", *enrolled,
			"error", err.Error())
		tx.Rollback()
		return
	}

	//set e-mail data
	data.CourseTitle = course.Title
	data.EventTitle = event.Title
	data.CourseID = course.ID
	data.User.ID = enrolled.UserID
	data.Status = enrolled.Status
	if err = data.User.Get(tx); err != nil {
		return
	}

	tx.Commit()
	return
}

func (enrolled *Enrolled) removeFromUnsubscribed(tx *sqlx.Tx) (err error) {

	//try to remove the user from the unsubscribed table
	_, err = tx.Exec(stmtDeleteUserFromUnsubscribed, enrolled.UserID, enrolled.EventID)
	if err != nil {
		log.Error("failed to remove user from unsubscribed table", "enrolled",
			*enrolled, "error", err.Error())
		tx.Rollback()
	}

	return
}

func (enrolled *Enrolled) hasEventStatus(tx *sqlx.Tx) (exists bool, err error) {

	stmt := stmtGetUserEnrolled
	if enrolled.Status == ONWAITLIST {
		stmt = stmtGetUserAtWaitList
	} else if enrolled.Status == UNSUBSCRIBED {
		stmt = stmtGetUserUnsubscribed
	}

	err = tx.Get(&exists, stmt, enrolled.EventID, enrolled.UserID)
	if err != nil {
		log.Error("failed to get if the user has the specified event status",
			"enrolled", *enrolled, "error", err.Error())
		tx.Rollback()
	}

	return
}

func (enrolled *Enrolled) updateStatus(tx *sqlx.Tx) (err error) {

	_, err = tx.Exec(stmtUpdateEnrollmentStatus, enrolled.UserID,
		enrolled.EventID, enrolled.Status)
	if err != nil {
		log.Error("failed to update user status", "enrolled",
			*enrolled, "error", err.Error())
		tx.Rollback()
	}

	return
}

func (enrolled *Enrolled) enroll(tx *sqlx.Tx) (err error) {

	_, err = tx.Exec(stmtEnrollUser, enrolled.UserID, enrolled.EventID,
		enrolled.Status)
	if err != nil {
		log.Error("failed to enroll user", "enrolled", *enrolled,
			"error", err.Error())
		tx.Rollback()
	}

	return
}

func (enrolled *Enrolled) unsubscribe(tx *sqlx.Tx) (err error) {

	//unsubscribe
	_, err = tx.Exec(stmtUnsubscribeUser, enrolled.UserID, enrolled.EventID)
	if err != nil {
		log.Error("failed to unsubscribe user", "enrolled", *enrolled,
			"error", err.Error())
		tx.Rollback()
		return
	}

	//insert into unsubscribed table
	_, err = tx.Exec(stmtInsertUserIntoUnsubscribed, enrolled.UserID, enrolled.EventID)
	if err != nil {
		log.Error("failed to remove user from unsubscribed table", "enrolled",
			*enrolled, "error", err.Error())
		tx.Rollback()
	}

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

	stmtGetUserEnrolled = `
		SELECT EXISTS (
			SELECT user_id
			FROM enrolled
			WHERE event_id = $1
				AND user_id = $2
				AND status != 1 /*on waitlist */
		) AS exists
	`

	stmtGetUserAtWaitList = `
		SELECT EXISTS (
			SELECT user_id
			FROM enrolled
			WHERE event_id = $1
				AND user_id = $2
				AND status = 1 /*on waitlist */
		) AS exists
	`

	stmtGetUserUnsubscribed = `
		SELECT EXISTS (
			SELECT user_id
			FROM unsubscribed
			WHERE event_id = $1
				AND user_id = $2
		) AS exists
	`

	stmtUpdateEnrollmentStatus = `
		UPDATE enrolled
		SET status = $3
		WHERE user_id = $1
			AND event_id = $2
	`

	stmtUserWillGetAutoEnrolled = `
		SELECT NOT EXISTS (
			SELECT en.user_id
			FROM enrolled en
			WHERE en.event_id = $1
				AND en.status = 1 /* on wait list */
				AND en.time_of_enrollment < (
					SELECT e.time_of_enrollment
					FROM enrolled e
					WHERE e.event_id = $1
						AND e.user_id = $2
				)
		) AS auto_enrolled
	`

	stmtUpdateStatus = `
		UPDATE enrolled
		SET status = $3
		WHERE event_id = $1
			AND user_id = $2
			AND status != 0
			AND status != 1
		RETURNING user_id
	`

	stmtSelectUserEnrollmentsExpired = `
		SELECT en.user_id, en.event_id, en.status, c.title AS course_title,
			e.title AS event_title,
		TO_CHAR (en.time_of_enrollment AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') AS time_of_enrollment
		FROM enrolled en JOIN events e ON en.event_id = e.id
			JOIN courses c ON e.course_id = c.id
		WHERE en.user_id = $1
			AND current_timestamp >= expiration_date
		ORDER BY time_of_enrollment DESC
	`

	stmtSelectUserEnrollments = `
		SELECT en.user_id, en.event_id,	en.status, c.title AS course_title,
			e.title AS event_title, c.id AS course_id,
			TO_CHAR (en.time_of_enrollment AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') AS time_of_enrollment
		FROM enrolled en JOIN events e ON en.event_id = e.id
			JOIN courses c ON e.course_id = c.id
		WHERE en.user_id = $1
			AND current_timestamp < expiration_date
		ORDER BY time_of_enrollment DESC
	`
)
