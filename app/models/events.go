package models

import (
	"database/sql"
	"turm/app"

	"github.com/jmoiron/sqlx"
	"github.com/revel/revel"
)

/*Event is a model of the event table. */
type Event struct {
	ID            int            `db:"id, primarykey, autoincrement"`
	CourseID      int            `db:"course_id"`
	Capacity      int            `db:"capacity"`
	HasWaitlist   bool           `db:"has_waitlist"`
	Title         string         `db:"title"`
	Annotation    sql.NullString `db:"annotation"`
	EnrollmentKey sql.NullString `db:"enrollment_key"`
	Meetings      Meetings       ``

	//Fullness is the number of users that enrolled in this event
	Fullness int ``
	//Percentage is (Fullness * 100) / Capacity
	Percentage int ``

	//used for enrollment
	EventStatus  EventStatus
	EnrollOption EnrollOption
	EnrollMsg    string
}

/*NewBlank creates a new blank event. */
func (event *Event) NewBlank(conf *EditEMailConfig) (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	err = tx.Get(event, stmtInsertBlankEvent, event.CourseID, event.Title)
	if err != nil {
		log.Error("failed to insert blank event", "event", event,
			"error", err.Error())
		tx.Rollback()
		return
	}

	//get edit notification e-mail data
	if err = conf.Get(tx); err != nil {
		return
	}

	tx.Commit()
	return
}

/*Update the specified column in the event table. */
func (event *Event) Update(tx *sqlx.Tx, column string, value interface{},
	conf *EditEMailConfig) (users EMailsData, err error) {

	txWasNil := (tx == nil)
	if txWasNil {
		tx, err = app.Db.Beginx()
		if err != nil {
			log.Error("failed to begin tx", "error", err.Error())
			return
		}
	}

	if err = updateByID(tx, column, "events", value, event.ID, event); err != nil {
		return
	}

	//get edit notification e-mail data
	if conf != nil {
		if err = conf.Get(tx); err != nil {
			return
		}
	}

	//if the capacity was changed and the user has a wait list
	//then get all users that get auto enrolled from that wait list
	if column == "capacity" {

		if err = event.GetColumnValue(tx, "has_waitlist"); err != nil {
			return
		}

		if event.HasWaitlist {

			if err = event.GetColumnValue(tx, "course_id"); err != nil {
				return
			}

			//get all information required for sending the e-mail
			course := Course{ID: event.CourseID}
			if err = course.GetColumnValue(tx, "title"); err != nil {
				return
			}
			if err = course.GetColumnValue(tx, "fee"); err != nil {
				return
			}

			status := ENROLLED
			if course.Fee.Valid {
				status = AWAITINGPAYMENT
			}

			//auto enroll users
			autoEnrollUsers := Users{}
			err = autoEnrollUsers.AutoEnrollFromWaitList(tx, &event.ID, status)
			if err != nil {
				return
			}

			if err = event.Get(tx); err != nil {
				return
			}

			//set e-mail data
			for _, user := range autoEnrollUsers {

				eMailUser := EMailData{
					CourseTitle: course.Title,
					CourseID:    course.ID,
					EventTitle:  event.Title,
					User:        user,
				}

				users = append(users, eMailUser)
			}
		}
	}

	if txWasNil {
		tx.Commit()
	}
	return
}

/*UpdateKey updates the enrollment key of an event. */
func (event *Event) UpdateKey() (err error) {

	err = app.Db.Get(event, stmtUpdateEnrollmentKey, event.EnrollmentKey, event.ID)
	if err != nil {
		log.Error("failed to update enrollment key", "event", *event,
			"error", err.Error())
	}
	return
}

/*UpdateWaitlist of an event. */
func (event *Event) UpdateWaitlist(option bool, v *revel.Validation) (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	//check if users are at the wait list
	if !option {
		var notEmpty bool
		if err = tx.Get(&notEmpty, stmtGetWaitlistEmpty, event.ID); err != nil {
			log.Error("failed to get if wait list ist empty", "event", *event,
				"error", err.Error())
			tx.Rollback()
			return
		} else if notEmpty {
			v.ErrorKey("validation.invalid.at.wait.list")
			tx.Commit()
			return
		}
	}

	//update the wait list
	if err = updateByID(tx, "has_waitlist", "events", option, event.ID, event); err != nil {
		return
	}

	tx.Commit()
	return
}

/*Delete an event. */
func (event *Event) Delete(v *revel.Validation) (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	//don't allow the deletion of events if users are enrolled in them
	var notEmpty bool
	if err = tx.Get(&notEmpty, stmtGetEventIsEmpty, event.ID); err != nil {
		log.Error("failed to get if the event is empty or not", "event",
			*event, "error", err.Error())
		tx.Rollback()
		return
	}
	if notEmpty {
		v.ErrorKey("validation.invalid.delete")
		tx.Commit()
		return
	}

	//don't allow courses to have no events and no calendar events
	course := Course{}
	if err = tx.Get(&course, stmtGetCourseIDOfEvent, event.ID); err != nil {
		log.Error("failed to get course id of event", "event", *event,
			"error", err.Error())
		tx.Rollback()
		return
	}

	//get course data for validation
	if err = course.GetForValidation(tx); err != nil {
		return
	}

	//validate changes
	if course.Active {
		if len(course.Events) > 0 {
			course.Events = course.Events[:len(course.Events)-1]
		}
		if course.Validate(v); v.HasErrors() {
			tx.Commit()
			return
		}
	}

	//delete event
	if err = deleteByID("id", "events", event.ID, tx); err != nil {
		return
	}

	tx.Commit()
	return
}

/*Duplicate an event. */
func (event *Event) Duplicate(tx *sqlx.Tx) (err error) {

	txWasNil := (tx == nil)
	if txWasNil {
		tx, err = app.Db.Beginx()
		if err != nil {
			log.Error("failed to begin tx", "error", err.Error())
			return
		}
	}

	var newID int
	err = tx.Get(&newID, stmtDuplicateEvent, event.CourseID, event.ID)
	if err != nil {
		log.Error("failed to duplicate event", "event", *event,
			"error", err.Error())
		tx.Rollback()
		return
	}

	//duplicate all meetings of this event
	if err = event.Meetings.Duplicate(tx, &newID, &event.ID); err != nil {
		return
	}

	if txWasNil {
		tx.Commit()
	}
	return
}

/*Get returns all data of one event. */
func (event *Event) Get(tx *sqlx.Tx) (err error) {

	err = tx.Get(event, stmtGetEvent, event.ID)
	if err != nil {
		log.Error("failed to get event by ID", "event", *event, "error", err.Error())
		tx.Rollback()
		return
	}

	//set the percentage field
	event.Percentage = (event.Fullness * 100) / event.Capacity
	return
}

/*GetColumnValue returns the value of a specific column. */
func (event *Event) GetColumnValue(tx *sqlx.Tx, column string) (err error) {

	return getColumnValue(tx, column, "events", event.ID, event)
}

//validateEnrollStatus sets the fields of the enroll status of an event
func (event *Event) validateEnrollStatus(tx *sqlx.Tx, userID *int,
	limit *sql.NullInt32) (err error) {

	enrollments := Enrollments{}
	if err = enrollments.SelectByCourse(tx, userID, &event.CourseID); err != nil {
		return
	}

	event.EventStatus.Full = (event.Capacity <= event.Fullness)

	for _, enrollment := range enrollments {

		//validate if the user already enrolled in this event
		if enrollment.EventID == event.ID {
			if enrollment.Status == ONWAITLIST {
				event.EventStatus.OnWaitlist = true
			} else {
				event.EventStatus.Enrolled = true
			}

		} else {

			//validate if the user already enrolled in other events of this course
			event.EventStatus.InOtherEvent = true
		}
	}

	//validate if the user already enrolled in another event and there are event limitations
	if limit.Valid {
		if len(enrollments) >= int(limit.Int32) {
			event.EventStatus.EnrollLimitReached = true
		}
	}

	return
}

//validateEnrollment validates whether a user can enroll in an event
func (event *Event) validateEnrollment(c *Course) {

	//all options disabling enrollment
	event.EnrollOption = NOENROLL

	if c.Expired || !c.Active {
		event.EnrollMsg = "validation.enrollment.not.active"
		return
	}
	if c.CourseStatus.AtBlacklist {
		event.EnrollMsg = "validation.enrollment.at.blacklist"
		return
	}
	if c.CourseStatus.NotLDAP {
		event.EnrollMsg = "validation.enrollment.no.ldap"
		return
	}
	if c.CourseStatus.NotSatisfyRestrictions {
		event.EnrollMsg = "validation.enrollment.not.satisfy.restrictions"
		return
	}
	if c.CourseStatus.MaxEnrollCoursesReached {
		if !event.EventStatus.Enrolled && !event.EventStatus.OnWaitlist {
			if !event.EventStatus.InOtherEvent {
				event.EnrollMsg = "validation.enrollment.max.enroll.reached"
				return
			}
		}
		c.CourseStatus.MaxEnrollCoursesReached = false
	}
	if event.EventStatus.EnrollLimitReached {
		if !event.EventStatus.Enrolled && !event.EventStatus.OnWaitlist {
			event.EnrollMsg = "validation.enrollment.limit.reached"
			return
		}
		event.EventStatus.EnrollLimitReached = false
	}

	//unsubscribe period is over
	if c.CourseStatus.UnsubscribeOver {
		if event.EventStatus.Enrolled || event.EventStatus.OnWaitlist {
			event.EnrollOption = NOUNSUBSCRIBE
		}
		event.EnrollMsg = "validation.enrollment.period.over"
		return
	}

	//enrollment period has not yet started
	if c.CourseStatus.NoEnrollmentPeriod {
		if !event.EventStatus.Enrolled && !event.EventStatus.OnWaitlist {
			event.EnrollMsg = "validation.enrollment.no.period"
			return
		}
	}

	//user is enrolled
	if event.EventStatus.Enrolled {
		event.EnrollOption = UNSUBSCRIBE
		return
	}

	//user is on wait list
	if event.EventStatus.OnWaitlist {
		event.EnrollOption = UNSUBSCRIBEFROMWAITLIST
		return
	}

	//full and no wait list
	if event.EventStatus.Full && !event.HasWaitlist {
		event.EnrollMsg = "validation.enrollment.full"
		return
	}

	//waitlist enrollment
	if event.EventStatus.Full && event.HasWaitlist {
		//not enrolled, course is full, not on wait list
		if !event.EventStatus.OnWaitlist {
			event.EnrollOption = ENROLLTOWAITLIST
			return
		}
	}

	event.EnrollOption = ENROLL
	return
}

/*Events holds all events of a course. */
type Events []Event

/*Get all events of a course. */
func (events *Events) Get(tx *sqlx.Tx, userID, courseID *int, manage bool,
	limit *sql.NullInt32) (err error) {

	txWasNil := (tx == nil)
	if txWasNil {
		tx, err = app.Db.Beginx()
		if err != nil {
			log.Error("failed to begin tx", "error", err.Error())
			return
		}
	}

	err = tx.Select(events, stmtSelectEvents, *courseID)
	if err != nil {
		log.Error("failed to get events of course", "course ID", *courseID, "error", err.Error())
		tx.Rollback()
		return
	}

	for key := range *events {

		//validate if the user is allowed to enroll in this event
		if !manage && *userID != 0 {
			if err = (*events)[key].validateEnrollStatus(tx, userID, limit); err != nil {
				return
			}
		}

		//get all meetings of this event
		(*events)[key].Percentage = ((*events)[key].Fullness * 100) / (*events)[key].Capacity
		if err = (*events)[key].Meetings.Get(tx, &(*events)[key].ID); err != nil {
			return
		}
	}

	if txWasNil {
		tx.Commit()
	}
	return
}

/*GetForValidation only returns event data required for course validation. */
func (events *Events) GetForValidation(tx *sqlx.Tx, courseID *int) (err error) {

	err = tx.Select(events, stmtSelectEvents, *courseID)
	if err != nil {
		log.Error("failed to get events of course for validation", "course ID", *courseID,
			"error", err.Error())
		tx.Rollback()
		return
	}

	for key := range *events {
		//get all meetings of this event
		(*events)[key].Percentage = ((*events)[key].Fullness * 100) / (*events)[key].Capacity
		if err = (*events)[key].Meetings.Get(tx, &(*events)[key].ID); err != nil {
			return
		}
	}

	return
}

/*Duplicate all events of a course. */
func (events *Events) Duplicate(tx *sqlx.Tx, courseIDNew, courseIDOld *int) (err error) {

	//get all event IDs
	err = tx.Select(events, stmtGetEventIDs, *courseIDOld)
	if err != nil {
		log.Error("failed to get all events for duplication", "course ID old",
			*courseIDOld, "error", err.Error())
		tx.Rollback()
		return
	}

	//duplicate each event
	for _, event := range *events {

		event.CourseID = *courseIDNew
		if err = event.Duplicate(tx); err != nil {
			return
		}
	}

	return
}

/*Insert all events of a course struct. */
func (events *Events) Insert(tx *sqlx.Tx, courseID *int) (err error) {

	for _, event := range *events {
		err = tx.Get(&event, stmtInsertEvent, event.Annotation, event.Capacity, *courseID,
			event.EnrollmentKey, event.HasWaitlist, event.Title)
		if err != nil {
			log.Error("failed to insert event of course", "course ID", *courseID,
				"error", err.Error())
			tx.Rollback()
			return
		}

		if err = event.Meetings.Insert(tx, &event.ID); err != nil {
			return
		}
	}
	return
}

const (
	stmtSelectEvents = `
		SELECT
			e.id, e.course_id, e.capacity, e.has_waitlist,
			e.title, e.annotation, e.enrollment_key,
			(
				SELECT COUNT(en.user_id)
				FROM enrolled en
				WHERE en.event_id = e.id
					AND status != 1 /*on waitlist*/
			) AS fullness
		FROM events e
		WHERE course_id = $1
		ORDER BY id ASC
	`

	stmtInsertBlankEvent = `
		INSERT INTO events (
				course_id, title, capacity, has_waitlist
			)
		VALUES (
				$1, $2, 1, false
		)
		RETURNING id, title
	`

	stmtDuplicateEvent = `
		INSERT INTO events
			(annotation, capacity, course_id, enrollment_key, has_waitlist, title)
		(
			SELECT
				annotation, capacity, $1 AS course_id, enrollment_key, has_waitlist, title
			FROM events
			WHERE id = $2
		)
		RETURNING id AS new_id
	`

	stmtGetEventIDs = `
		SELECT id
		FROM events
		WHERE course_id = $1
	`

	stmtInsertEvent = `
		INSERT INTO events
			(annotation, capacity, course_id, enrollment_key, has_waitlist, title)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	stmtUpdateEnrollmentKey = `
		UPDATE events
		SET enrollment_key = CRYPT($1, gen_salt('bf'))
		WHERE id = $2
		RETURNING id
	`

	stmtGetEvent = `
		SELECT
			e.id, e.course_id, e.capacity, e.has_waitlist,
			e.title, e.annotation, e.enrollment_key,
			(
				SELECT COUNT(en.user_id)
				FROM enrolled en
				WHERE en.event_id = e.id
					AND status != 1 /*on waitlist*/
			) AS fullness
		FROM events e
		WHERE id = $1
	`

	stmtGetEventVisible = `
		SELECT c.visible
		FROM courses c JOIN events e ON c.id = e.course_id
		WHERE e.id = $1
	`

	stmtGetCourseIDByEvent = `
		SELECT course_id AS id
		FROM events
		WHERE id = $1
	`

	stmtGetEventIsEmpty = `
		SELECT EXISTS (
			SELECT true
			FROM enrolled
			WHERE event_id = $1
		) AS not_empty
	`

	stmtGetWaitlistEmpty = `
		SELECT EXISTS (
			SELECT true
			FROM enrolled
			WHERE event_id = $1
				AND status = 1 /*on waitlist*/
		)
	`

	stmtGetCourseIDOfEvent = `
		SELECT course_id AS id
		FROM events e
		WHERE e.id = $1
	`
)
