package models

import (
	"database/sql"
	"encoding/json"
	"regexp"
	"strconv"
	"time"
	"turm/app"

	"github.com/jmoiron/sqlx"
	"github.com/revel/revel"
)

/*Course is a model of the course table. */
type Course struct {
	ID                int             `db:"id, primarykey, autoincrement"`
	Title             string          `db:"title"`
	Creator           sql.NullInt32   `db:"creator"`
	Subtitle          sql.NullString  `db:"subtitle"`
	Visible           bool            `db:"visible"`
	Active            bool            `db:"active"`
	OnlyLDAP          bool            `db:"only_ldap"`
	CreationDate      string          `db:"creation_date"`
	Description       sql.NullString  `db:"description"`
	Speaker           sql.NullString  `db:"speaker"`
	Fee               sql.NullFloat64 `db:"fee"`
	CustomEMail       sql.NullString  `db:"custom_email"`
	EnrollLimitEvents sql.NullInt32   `db:"enroll_limit_events"`
	EnrollmentStart   string          `db:"enrollment_start"`
	EnrollmentEnd     string          `db:"enrollment_end"`
	UnsubscribeEnd    sql.NullString  `db:"unsubscribe_end"`
	ExpirationDate    string          `db:"expiration_date"`
	ParentID          sql.NullInt32   `db:"parent_id"`

	//course data of different tables
	Events         Events         ``
	CalendarEvents CalendarEvents ``
	Editors        UserList       ``
	Instructors    UserList       ``
	Blacklist      UserList       ``
	Whitelist      UserList       ``
	Restrictions   Restrictions   ``

	//additional information required when displaying the course
	CreatorData User ``
	//path to the course entry in the groups tree
	Path Groups ``
	//used for correct template rendering
	CreatorID string
	Expired   bool

	//used to add/edit course restrictions
	CoursesOfStudies CoursesOfStudies
	Degrees          Degrees

	//used for enrollment
	CourseStatus CourseStatus
	Manage       bool
}

/*Validate all Course fields. */
func (course *Course) Validate(v *revel.Validation) {

	now := time.Now().Format(revel.TimeFormats[0])

	//now < EnrollmentStart
	if now >= course.EnrollmentStart {
		v.ErrorKey("validation.invalid.enrollment.start")
	}

	//EnrollmentStart < EnrollmentEnd
	if course.EnrollmentStart >= course.EnrollmentEnd {
		v.ErrorKey("validation.invalid.enrollment.end")
	}

	if course.UnsubscribeEnd.Valid {
		//if UnsubscribeEnd, then EnrollmentEnd <= UnsubscribeEnd
		if course.EnrollmentEnd > course.UnsubscribeEnd.String {
			v.ErrorKey("validation.invalid.unsubscribe.end")
		}
		//if UnsubscribeEnd, then UnsubscribeEnd <= ExpirationDate
		if course.ExpirationDate < course.UnsubscribeEnd.String {
			v.ErrorKey("validation.invalid.unsubscribe.expiration")
		}
	}

	//EnrollmentEnd <= ExpirationDate
	if course.EnrollmentEnd > course.ExpirationDate {
		v.ErrorKey("validation.invalid.expiration.date")
	}

	//ParentID not null
	v.Required(course.ParentID.Valid).
		MessageKey("validation.invalid.parent")

	//for all meetings
	for _, event := range course.Events {
		for _, meeting := range event.Meetings {

			//EnrollmentStart <= MeetingStart
			if course.EnrollmentStart > meeting.MeetingStart {
				v.ErrorKey("validation.invalid.meeting.start")
			}

			//MeetingStart < MeetingEnd
			if meeting.MeetingStart >= meeting.MeetingEnd {
				v.ErrorKey("validation.invalid.meeting.end")
			}
		}
	}

	if len(course.Events) == 0 {
		v.ErrorKey("validation.invalid.len.events")
	}
}

/*Update the specified column in the course table. */
func (course *Course) Update(tx *sqlx.Tx, column string, value interface{}) (err error) {
	return updateByID(tx, column, "courses", value, course.ID, course)
}

/*Get all course data. If manage is false, only get publicly available course
data. Also, if it is false, get enrollment information for this user for each
event. */
func (course *Course) Get(tx *sqlx.Tx, manage bool, userID int) (err error) {

	txWasNil := (tx == nil)
	if txWasNil {
		tx, err = app.Db.Beginx()
		if err != nil {
			log.Error("failed to begin tx", "error", err.Error())
			return
		}
	}

	course.Manage = manage

	//get general course data
	err = tx.Get(course, stmtSelectCourse, course.ID, app.TimeZone)
	if err != nil {
		log.Error("failed to get course", "course ID", course.ID, "error", err.Error())
		tx.Rollback()
		return
	}

	//get additional fields
	if err = course.Editors.Get(tx, &course.ID, "editors"); err != nil {
		return
	}
	if err = course.Instructors.Get(tx, &course.ID, "instructors"); err != nil {
		return
	}
	if err = course.Blacklist.Get(tx, &course.ID, "blacklists"); err != nil {
		return
	}
	if err = course.Whitelist.Get(tx, &course.ID, "whitelists"); err != nil {
		return
	}
	if err = course.Restrictions.Get(tx, &course.ID); err != nil {
		return
	}

	if !manage && userID != 0 {
		if err = course.validateEnrollment(tx, userID); err != nil {
			return
		}
	}

	//get the events of the course
	if err = course.Events.Get(tx, &userID, &course.ID, manage, &course.EnrollLimitEvents); err != nil {
		return
	}

	//get the calander events of a course
	if err = course.CalendarEvents.Get(tx, &course.ID, day); err != nil {
		return
	}

	if manage {
		//get courses of studies and degrees
		if err = course.CoursesOfStudies.Get(tx); err != nil {
			return
		}
		if err = course.Degrees.Get(tx); err != nil {
			return
		}
	}

	//get enroll information for each event
	if !manage {
		for key := range course.Events {
			course.Events[key].validateEnrollment(course)
		}
	}

	//get more detailed creator data
	if course.Creator.Valid {
		course.CreatorData.ID = int(course.Creator.Int32)
		if err = course.CreatorData.GetBasicData(tx); err != nil {
			return
		}
	}
	course.CreatorID = strconv.Itoa(int(course.Creator.Int32))

	//get the path of the course in the groups tree
	if err = course.Path.SelectPath(&course.ID, tx); err != nil {
		return
	}

	//reset some data
	if manage {
		course.Blacklist = UserList{}
		course.Whitelist = UserList{}
	}

	if txWasNil {
		tx.Commit()
	}
	return
}

/*GetForEnrollment returns only the information required for enrollment. */
func (course *Course) GetForEnrollment(tx *sqlx.Tx, userID, eventID *int) (err error) {

	//get general course data
	err = tx.Get(course, stmtSelectCourse, course.ID, app.TimeZone)
	if err != nil {
		log.Error("failed to get course", "course ID", course.ID, "error", err.Error())
		tx.Rollback()
		return
	}

	if err = course.Blacklist.Get(tx, &course.ID, "blacklists"); err != nil {
		return
	}
	if err = course.Whitelist.Get(tx, &course.ID, "whitelists"); err != nil {
		return
	}
	if err = course.Restrictions.Get(tx, &course.ID); err != nil {
		return
	}

	err = course.validateEnrollment(tx, *userID)
	return
}

//validateEnrollment validates whether a user can enroll in a course
func (course *Course) validateEnrollment(tx *sqlx.Tx, userID int) (err error) {

	//if the user is at the blacklist
	for _, user := range course.Blacklist {
		if user.UserID == userID {
			course.CourseStatus.AtBlacklist = true
			return
		}
	}

	//validate if the user is at the whitelist
	for _, user := range course.Whitelist {
		if user.UserID == userID {
			course.CourseStatus.AtWhitelist = true
		}
	}

	//skip some validation if the user is at the whitelist
	if !course.CourseStatus.AtWhitelist {

		//validate if the user complies with the course restrictions
		//therefore, first get more user information
		user := User{ID: userID}
		if err = user.Get(tx); err != nil {
			return
		}

		//validate the NotLDAP field
		if course.OnlyLDAP && !user.IsLDAP {
			course.CourseStatus.NotLDAP = true
			return
		}

		//validate if the user complies with specific restrictions
		complies := false
		for _, restriction := range course.Restrictions {

			for _, value := range user.Studies {

				//validate degree
				if restriction.DegreeID.Valid {
					if restriction.DegreeID.Int64 != int64(value.DegreeID) {
						continue
					}
				}

				//validate studies
				if restriction.CourseOfStudiesID.Valid {
					if restriction.CourseOfStudiesID.Int64 != int64(value.CourseOfStudiesID) {
						continue
					}
				}

				//validate minimum semester
				if restriction.MinimumSemester.Valid {
					if restriction.MinimumSemester.Int64 > int64(value.Semester) {
						continue
					}
				}

				complies = true
				break
			}

			if complies {
				break
			}
		}
		if !complies && len(course.Restrictions) > 0 {
			course.CourseStatus.NotSatisfyRestrictions = true
			return
		}
	}

	//validate if the enrollment period is active
	if err = tx.Get(&course.CourseStatus, stmtValidateEnrollmentPeriod, course.ID); err != nil {
		log.Error("failed to validate the enrollment period", "courseID", course.ID,
			"err", err.Error())
		tx.Rollback()
		return
	}
	if course.CourseStatus.NoEnrollmentPeriod {
		return
	}

	//get the course enrollment limit for this course
	err = tx.Get(&course.CourseStatus.MaxEnrollCourses, stmtParentsGetCourseLimit,
		course.ParentID)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Error("failed to retrieve information for this parent's course limit",
				"parentID", course.ParentID, "error", err.Error())
			tx.Rollback()
			return
		}
		err = nil
	}

	if course.CourseStatus.MaxEnrollCourses != 0 {
		//validate if the user reached the maximum enroll limit
		var enrollments int
		err = tx.Get(&enrollments, stmtGetCountUserEnrollments, course.ParentID, userID)
		if err != nil {
			log.Error("failed to count user enrollments", "parentID", course.ParentID,
				"userID", userID, "error", err.Error())
			tx.Rollback()
			return
		}

		if course.CourseStatus.MaxEnrollCourses <= enrollments {
			course.CourseStatus.MaxEnrollCoursesReached = true
		}
	}
	return
}

/*NewBlank creates a new blank course. */
func (course *Course) NewBlank(creatorID *int, title *string) (err error) {

	now := time.Now().Format(revel.TimeFormats[0])

	err = app.Db.Get(course, stmtInsertBlankCourse, now, *creatorID, *title)
	if err != nil {
		log.Error("failed to insert blank course", "now", now,
			"creator ID", *creatorID, "error", err.Error())
	}
	return
}

/*Delete a course. Courses must be inactive or expired to be deleted. */
func (course *Course) Delete() (valid bool, err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	err = tx.Get(&valid, stmtCourseIsInactiveOrExpired, course.ID)
	if err != nil {
		log.Error("failed to get validity of course deletion", "course ID", course.ID,
			"error", err.Error())
		tx.Rollback()
		return
	}

	if valid {
		if err = deleteByID("id", "courses", course.ID, tx); err != nil {
			return
		}
	}

	tx.Commit()
	return
}

/*Activate a course. */
func (course *Course) Activate(v *revel.Validation) (invalid bool, err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	if err = course.Get(tx, true, 0); err != nil {
		return
	}

	if course.Validate(v); v.HasErrors() {
		invalid = true
		return
	}

	if err = course.Update(tx, "active", true); err != nil {
		return
	}

	tx.Commit()
	return
}

/*Duplicate a course. */
func (course *Course) Duplicate() (err error) {

	now := time.Now().Format(revel.TimeFormats[0])
	courseIDOld := course.ID

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	//duplicate general course data
	err = tx.Get(course, stmtDuplicateCourse, course.ID, course.Title, now)
	if err != nil {
		log.Error("failed to duplicate course", "course ID", course.ID, "title",
			course.Title, "now", now, "error", err.Error())
		tx.Rollback()
		return
	}

	//duplicate events and meetings
	if err = course.Events.Duplicate(tx, &course.ID, &courseIDOld); err != nil {
		return
	}

	//duplicate user lists
	if err = course.Editors.Duplicate(tx, &course.ID, &courseIDOld, "editors"); err != nil {
		return
	}
	if err = course.Instructors.Duplicate(tx, &course.ID, &courseIDOld, "instructors"); err != nil {
		return
	}
	if err = course.Whitelist.Duplicate(tx, &course.ID, &courseIDOld, "whitelists"); err != nil {
		return
	}
	if err = course.Blacklist.Duplicate(tx, &course.ID, &courseIDOld, "blacklists"); err != nil {
		return
	}

	//TODO: duplicate restrictions

	tx.Commit()
	return
}

/*Load a course from a JSON file. The JSON can have the struct of the old Turm2. */
func (course *Course) Load(oldStruct bool, data *[]byte) (success bool, err error) {

	if !oldStruct {
		//unmarshal into the course struct
		err = json.Unmarshal(*data, &course)
		if err != nil {
			log.Error("failed to unmarshal into new struct", "data",
				*data, "error", err.Error())
			return
		}

	} else {
		//unmarshal the struct into the old layout
		//then transfer the data to the new course struct
		//TODO
	}

	return
}

/*Insert a new course from a provided course struct. */
func (course *Course) Insert(creatorID *int, title *string) (err error) {

	now := time.Now().Format(revel.TimeFormats[0])

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	err = tx.Get(course, stmtInsertCourse, now, *creatorID, course.CustomEMail, course.Description,
		course.EnrollLimitEvents, course.EnrollmentEnd, course.EnrollmentStart, course.ExpirationDate,
		course.Fee, course.OnlyLDAP, course.Speaker, course.Subtitle, title, course.UnsubscribeEnd, course.Visible)
	if err != nil {
		log.Error("failed to insert general course data", "creator ID", *creatorID,
			"title", *title, "now", now, "course", *course, "error", err.Error())
		tx.Rollback()
		return
	}

	if err = course.Events.Insert(tx, &course.ID); err != nil {
		return
	}
	if err = course.Editors.Insert(tx, &course.ID, "editors"); err != nil {
		return
	}
	if err = course.Instructors.Insert(tx, &course.ID, "instructors"); err != nil {
		return
	}
	if err = course.Blacklist.Insert(tx, &course.ID, "blacklists"); err != nil {
		return
	}
	if err = course.Whitelist.Insert(tx, &course.ID, "whitelists"); err != nil {
		return
	}

	//TODO: insert restrictions

	tx.Commit()
	return
}

//FeePattern is the regular expression of accepted course fees
var FeePattern = regexp.MustCompile("^([0-9]{1,6}([,|.][0-9]{0,2})?)?")

const (
	stmtSelectCourse = `
		SELECT
			id, title, creator, subtitle, visible, active, only_ldap, parent_id,
			description, fee, custom_email, enroll_limit_events, speaker,
			TO_CHAR (creation_date AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') AS creation_date,
			TO_CHAR (enrollment_start AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') AS enrollment_start,
			TO_CHAR (enrollment_end AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') AS enrollment_end,
			TO_CHAR (unsubscribe_end AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') AS unsubscribe_end,
			TO_CHAR (expiration_date AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') AS expiration_date,
			(current_timestamp >= expiration_date) AS expired
		FROM courses
		WHERE id = $1
	`

	stmtInsertBlankCourse = `
		INSERT INTO courses (
			title, creator, visible, active, only_ldap, creation_date,
			enrollment_start, enrollment_end, expiration_date
		)
		VALUES (
			$3, $2, false, false, false, $1, $1, $1, $1
		)
		RETURNING id, title
	`

	stmtCourseIsInactiveOrExpired = `
		SELECT true AS valid
		FROM courses
		WHERE id = $1
			AND (
				active = false
				OR
				(current_timestamp > expiration_date)
			)
	`

	stmtDuplicateCourse = `
		INSERT INTO courses (
			title, subtitle, active, creation_date, creator, custom_email, description, enroll_limit_events, enrollment_end,
			enrollment_start, expiration_date, fee, only_ldap, parent_id, speaker, unsubscribe_end, visible
		)
		(
			SELECT
					$2 AS title, subtitle, active, $3 AS creation_date, creator, custom_email, description, enroll_limit_events, enrollment_end,
					enrollment_start, expiration_date, fee, only_ldap, parent_id, speaker, unsubscribe_end, visible
			FROM courses
			WHERE id = $1
		)
		RETURNING id, title
	`

	stmtInsertCourse = `
		INSERT INTO courses
			(active, creation_date, creator, custom_email, description, enroll_limit_events, enrollment_end, enrollment_start,
			expiration_date, fee, only_ldap, speaker, subtitle, title, unsubscribe_end, visible)
		VALUES
			(false, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, title
	`

	stmtValidateEnrollmentPeriod = `
		SELECT
			(current_timestamp < enrollment_start OR
				current_timestamp > enrollment_end) AS no_enrollment_period,
			(current_timestamp > unsubscribe_end AND
				unsubscribe_end IS NOT NULL) AS unsubscribe_over
		FROM courses
		WHERE id = $1
	`
)
