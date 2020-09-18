package models

import (
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
)

func (status EnrollmentStatus) String() string {
	return [...]string{"enrolled", "on waitlist", "awaiting payment", "paid", "freed"}[status]
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

/*Validate Enrolled fields. */
func (enrolled *Enrolled) Validate(v *revel.Validation) {
	//TODO
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

/*Validate Unsubscribed fields. */
func (unsubscribed *Unsubscribed) Validate(v *revel.Validation) {
	//TODO
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
				/* starting entries */
				SELECT id, parent_id, course_limit
				FROM groups
				WHERE parent_id = (

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
)
