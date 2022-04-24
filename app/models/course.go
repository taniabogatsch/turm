package models

import (
	"database/sql"
	"encoding/json"
	"errors"
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
	CreationDate      time.Time       `db:"creation_date"`
	Description       sql.NullString  `db:"description"`
	Speaker           sql.NullString  `db:"speaker"`
	Fee               sql.NullFloat64 `db:"fee"`
	CustomEMail       sql.NullString  `db:"custom_email"`
	EnrollLimitEvents sql.NullInt32   `db:"enroll_limit_events"`
	EnrollmentStart   time.Time       `db:"enrollment_start"`
	EnrollmentEnd     time.Time       `db:"enrollment_end"`
	UnsubscribeEnd    sql.NullTime    `db:"unsubscribe_end"`
	ExpirationDate    time.Time       `db:"expiration_date"`
	ParentID          sql.NullInt32   `db:"parent_id"`

	//course data of different tables
	Events         Events         ``
	CalendarEvents CalendarEvents ``
	Editors        UserList       ``
	Instructors    UserList       ``
	Blocklist      UserList       ``
	Allowlist      UserList       ``
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

	//used to render buttons for redirect
	CanEdit               bool `db:"can_edit"`
	CanManageParticipants bool `db:"can_manage_participants"`
	IsCreator             bool

	//used for pretty timestamp rendering
	CreationDateStr    string         `db:"creation_date_str"`
	EnrollmentStartStr string         `db:"enrollment_start_str"`
	EnrollmentEndStr   string         `db:"enrollment_end_str"`
	UnsubscribeEndStr  sql.NullString `db:"unsubscribe_end_str"`
	ExpirationDateStr  string         `db:"expiration_date_str"`
}

/*Validate all course fields. */
func (course *Course) Validate(v *revel.Validation) {

	now := time.Now()

	if !course.Active {
		//now < EnrollmentStart
		if now.After(course.EnrollmentStart) {
			v.ErrorKey("validation.invalid.enrollment.start")
		}
	}

	//EnrollmentStart < EnrollmentEnd
	if course.EnrollmentStart.After(course.EnrollmentEnd) {
		v.ErrorKey("validation.invalid.enrollment.end")
	}

	if course.UnsubscribeEnd.Valid {
		//if UnsubscribeEnd, then EnrollmentEnd <= UnsubscribeEnd
		if course.EnrollmentEnd.After(course.UnsubscribeEnd.Time) {
			v.ErrorKey("validation.invalid.unsubscribe.end")
		}
		//if UnsubscribeEnd, then UnsubscribeEnd <= ExpirationDate
		if course.ExpirationDate.Before(course.UnsubscribeEnd.Time) {
			v.ErrorKey("validation.invalid.unsubscribe.expiration")
		}
	}

	//EnrollmentEnd <= ExpirationDate
	if course.EnrollmentEnd.After(course.ExpirationDate) {
		v.ErrorKey("validation.invalid.expiration.date")
	}

	//ParentID not null
	v.Required(course.ParentID.Valid).
		MessageKey("validation.invalid.parent")

	//for all meetings
	for _, event := range course.Events {
		for _, meeting := range event.Meetings {

			//EnrollmentStart <= MeetingStart
			if course.EnrollmentStart.After(meeting.MeetingStart) {
				v.ErrorKey("validation.invalid.meeting.start")
			}

			//MeetingStart < MeetingEnd
			if meeting.MeetingStart.After(meeting.MeetingEnd) {
				v.ErrorKey("validation.invalid.meeting.end")
			}
		}
	}

	if len(course.Events) == 0 && len(course.CalendarEvents) == 0 {
		v.ErrorKey("validation.invalid.len.events")
	}
}

/*GetVisible of a course. */
func (course *Course) GetVisible(elem string) (err error) {

	switch elem {
	case "course":
		err = course.GetColumnValue(nil, "visible")
	case "event":
		err = app.Db.Get(course, stmtGetEventVisible, course.ID)
	default:
		err = errors.New("invalid parameter type")
	}

	if err != nil {
		log.Error("failed to get if course is visible", "course", *course,
			"elem", elem, "error", err.Error())
	}
	return
}

/*Update the specified column in the course table. */
func (course *Course) Update(tx *sqlx.Tx, column string, value interface{},
	conf *EditEMailConfig) (err error) {

	txWasNil := (tx == nil)
	if txWasNil {
		tx, err = app.Db.Beginx()
		if err != nil {
			log.Error("failed to begin tx", "error", err.Error())
			return
		}
	}

	//change the status of users enrolled in this course if a fee is added/deleted
	if column == "fee" {

		fee, ok := value.(sql.NullFloat64)
		if !ok {

			_, ok := value.(sql.NullString)
			if !ok {
				err = errors.New("parsing error")
				log.Error("failed to parse fee from interface", "value", value,
					"error", err.Error())
				tx.Rollback()
				return
			}
			fee.Valid = false
		}

		if err = course.GetColumnValue(tx, "fee"); err != nil {
			return
		}

		updateStatus := false
		status := ENROLLED

		//a fee is added
		if fee.Valid && !course.Fee.Valid {
			updateStatus = true
			status = AWAITINGPAYMENT
		}
		//a fee is deleted
		if !fee.Valid && course.Fee.Valid {
			updateStatus = true
		}

		if updateStatus {
			_, err = tx.Exec(stmtUpdateEnrollmentStatusDueToFee, course.ID, status)
			if err != nil {
				log.Error("failed to update enrollment status due to fee", "courseID",
					course.ID, "status", status, "error", err.Error())
				tx.Rollback()
				return
			}
		}
	}

	//update the course field
	if err = updateByID(tx, column, "courses", value, course.ID, course); err != nil {
		return
	}

	//get edit notification e-mail data
	if conf != nil {
		if err = conf.Get(tx); err != nil {
			return
		}
	}

	if txWasNil {
		tx.Commit()
	}
	return
}

/*UpdateTimestamp of a course. Also ensures validitiy, if the course is already active. */
func (course *Course) UpdateTimestamp(v *revel.Validation, conf *EditEMailConfig,
	fieldID string, t time.Time, valid bool) (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	//get course data for validation
	if err = course.GetForValidation(tx); err != nil {
		return
	}

	//validate changes
	if course.Active {

		switch fieldID {
		case "enrollment_start":
			course.EnrollmentStart = t
		case "enrollment_end":
			course.EnrollmentEnd = t
		case "unsubscribe_end":
			course.UnsubscribeEnd = sql.NullTime{t, valid}
		case "expiration_date":
			course.ExpirationDate = t
		}

		if course.Validate(v); v.HasErrors() {
			tx.Commit()
			return
		}
	}

	//no errors, update the course
	if fieldID == "unsubscribe_end" {
		if err = course.Update(tx, fieldID, sql.NullTime{t, valid}, conf); err != nil {
			return
		}
	} else {
		if err = course.Update(tx, fieldID, t, conf); err != nil {
			return
		}
	}

	tx.Commit()
	return
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
	err = tx.Get(course, stmtGetCourse, course.ID, app.TimeZone)
	if err != nil {
		log.Error("failed to get course", "course ID", course.ID, "error", err.Error())
		tx.Rollback()
		return
	}

	//get if the user is allowed to edit the course and to manage the participants
	if int(course.Creator.Int32) == userID {
		course.CanEdit = true
		course.CanManageParticipants = true
	} else {
		err = tx.Get(course, stmtIsEditorInstructorOfCourse, userID, course.ID)
		if err != nil {
			log.Error("failed to get if user is editor or instructor of the course",
				"userID", userID, "courseID", course.ID, "error", err.Error())
			tx.Rollback()
			return
		}
	}

	//get additional fields
	if err = course.Editors.Get(tx, &course.ID, "editors"); err != nil {
		return
	}
	if err = course.Instructors.Get(tx, &course.ID, "instructors"); err != nil {
		return
	}
	if err = course.Blocklist.Get(tx, &course.ID, "blocklists"); err != nil {
		return
	}
	if err = course.Allowlist.Get(tx, &course.ID, "allowlists"); err != nil {
		return
	}
	if err = course.Restrictions.Get(tx, &course.ID); err != nil {
		return
	}

	//get the events of the course
	err = course.Events.Get(tx, &userID, &course.ID, manage, &course.EnrollLimitEvents)
	if err != nil {
		return
	}

	now := time.Now()
	weekday := time.Now().Weekday()
	monday := now.AddDate(0, 0, -1*(int(weekday)-1))

	//get the calender events of a course
	err = course.CalendarEvents.Get(tx, &course.ID, monday, userID)
	if err != nil {
		return
	}

	if manage {
		//get all courses of studies
		if err = course.CoursesOfStudies.Get(tx); err != nil {
			return
		}
		//get all degrees
		if err = course.Degrees.Get(tx); err != nil {
			return
		}
	}

	//get course data for enrollment validation
	if !manage && userID != 0 {
		if err = course.validateEnrollment(tx, userID); err != nil {
			return
		}
	}

	//get enroll information for each event
	if !manage {
		for key := range course.Events {
			course.Events[key].validateEnrollment(course)
		}
	}

	//get enroll information for each calendar event
	if !manage {
		for key := range course.CalendarEvents {
			err = course.CalendarEvents[key].validateEnrollment(tx, course, userID)
			if err != nil {
				return
			}
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
	if !manage {
		course.Blocklist = UserList{}
		course.Allowlist = UserList{}
	}

	if txWasNil {
		tx.Commit()
	}
	return
}

/*GetForValidation only returns the course data required for validation. */
func (course *Course) GetForValidation(tx *sqlx.Tx) (err error) {

	//get general course data
	err = tx.Get(course, stmtGetCourse, course.ID, app.TimeZone)
	if err != nil {
		log.Error("failed to get course for validation", "course ID", course.ID,
			"error", err.Error())
		tx.Rollback()
		return
	}

	//get the events of the course
	err = course.Events.GetForValidation(tx, &course.ID)
	if err != nil {
		return
	}

	now := time.Now()
	weekday := time.Now().Weekday()
	monday := now.AddDate(0, 0, -1*(int(weekday)-1))

	//get the calender events of a course
	//TODO: use more efficient function (GetForValidation)
	err = course.CalendarEvents.Get(tx, &course.ID, monday, 0)
	if err != nil {
		return
	}

	return
}

/*GetForEnrollment returns only the information required for enrollment. */
func (course *Course) GetForEnrollment(tx *sqlx.Tx, userID, eventID *int) (err error) {

	//get general course data
	err = tx.Get(course, stmtGetCourse, course.ID, app.TimeZone)
	if err != nil {
		log.Error("failed to get course", "course ID", course.ID, "error", err.Error())
		tx.Rollback()
		return
	}

	if err = course.Blocklist.Get(tx, &course.ID, "blocklists"); err != nil {
		return
	}
	if err = course.Allowlist.Get(tx, &course.ID, "allowlists"); err != nil {
		return
	}
	if err = course.Restrictions.Get(tx, &course.ID); err != nil {
		return
	}

	err = course.validateEnrollment(tx, *userID)
	return
}

/*GetColumnValue returns the value of a specific column. */
func (course *Course) GetColumnValue(tx *sqlx.Tx, column string) (err error) {

	return getColumnValue(tx, column, "courses", course.ID, course)
}

//validateEnrollment validates whether a user can enroll in a course
func (course *Course) validateEnrollment(tx *sqlx.Tx, userID int) (err error) {

	//if the user is at the blocklist
	for _, user := range course.Blocklist {
		if user.UserID == userID {
			course.CourseStatus.AtBlocklist = true
			return
		}
	}

	//validate if the user is at the allowlist
	for _, user := range course.Allowlist {
		if user.UserID == userID {
			course.CourseStatus.AtAllowlist = true
		}
	}

	//skip some validation if the user is at the allowlist
	if !course.CourseStatus.AtAllowlist {

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
func (course *Course) NewBlank() (err error) {

	err = app.Db.Get(course, stmtInsertBlankCourse, course.Creator, course.Title)
	if err != nil {
		log.Error("failed to insert blank course", "creator ID", course.Creator,
			"title", course.Title, "error", err.Error())
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

	if err = course.GetColumnValue(tx, "active"); err != nil {
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
func (course *Course) Activate(v *revel.Validation) (invalid bool,
	users EMailsData, err error) {

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

	//update the active flag
	conf := EditEMailConfig{}
	if err = course.Update(tx, "active", true, &conf); err != nil {
		return
	}

	//set e-mail data for each editor
	for _, editor := range course.Editors {

		data := EMailData{
			CourseTitle: course.Title,
			CourseID:    course.ID,
			CourseRole:  "editors",
			ViewMatrNr:  editor.ViewMatrNr,
		}
		data.User.ID = editor.UserID
		if err = data.User.Get(tx); err != nil {
			return
		}

		users = append(users, data)
	}

	//set e-mail data for each instructor
	for _, instructor := range course.Instructors {

		data := EMailData{
			CourseTitle: course.Title,
			CourseID:    course.ID,
			CourseRole:  "instructors",
			ViewMatrNr:  instructor.ViewMatrNr,
		}
		data.User.ID = instructor.UserID
		if err = data.User.Get(tx); err != nil {
			return
		}

		users = append(users, data)
	}

	tx.Commit()
	return
}

/*Duplicate a course. */
func (course *Course) Duplicate() (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	courseIDOld := course.ID

	//duplicate general course data
	err = tx.Get(course, stmtDuplicateCourse, course.ID, course.Title, course.Creator)
	if err != nil {
		log.Error("failed to duplicate course", "course ID", course.ID, "title",
			course.Title, "creator", course.Creator, "error", err.Error())
		tx.Rollback()
		return
	}

	//duplicate events and meetings
	if err = course.Events.Duplicate(tx, &course.ID, &courseIDOld); err != nil {
		return
	}

	//duplicate calendar events
	if err = course.CalendarEvents.Duplicate(tx, &course.ID, &courseIDOld); err != nil {
		return
	}

	//duplicate user lists
	if err = course.Editors.Duplicate(tx, &course.ID, &courseIDOld, "editors"); err != nil {
		return
	}
	if err = course.Instructors.Duplicate(tx, &course.ID, &courseIDOld, "instructors"); err != nil {
		return
	}
	if err = course.Allowlist.Duplicate(tx, &course.ID, &courseIDOld, "allowlists"); err != nil {
		return
	}
	if err = course.Blocklist.Duplicate(tx, &course.ID, &courseIDOld, "blocklists"); err != nil {
		return
	}

	//duplicate restrictions
	if err = course.Restrictions.Duplicate(tx, &course.ID, &courseIDOld); err != nil {
		return
	}

	tx.Commit()
	return
}

/*Load a course from a JSON file. The JSON can have the struct of the old Turm2. */
func (course *Course) Load(version int, data *[]byte) (success bool, err error) {

	if version == 4 {
		//unmarshal into the course struct
		err = json.Unmarshal(*data, &course)
		if err != nil {
			log.Error("failed to unmarshal into new struct", "data",
				*data, "error", err.Error())
			return
		}

	} else if version == 1 || version == 2 {
		//unmarshal the struct into the version 2 layout
		version2Course := Version2Course{}
		err = json.Unmarshal(*data, &version2Course)
		if err != nil {
			log.Error("failed to unmarshal into version 2 struct", "data",
				*data, "error", err.Error())
			return
		}

		//then transform the data to the current (version 4) course struct
		err = version2Course.Transform(course)

	} else if version == 3 {
		//unmarshal the struct into the version 3 layout
		version3Course := Version3Course{}
		err = json.Unmarshal(*data, &version3Course)
		if err != nil {
			log.Error("failed to unmarshal into version 3 struct", "data",
				*data, "error", err.Error())
			return
		}

		//then transform the data to the current (version 4) course struct
		version3Course.Transform(course)
	}

	return
}

/*InsertUploadedCourse a new course from a provided course struct. The course
struct is extracted from an uploaded JSON file. */
func (course *Course) InsertUploadedCourse() (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	err = tx.Get(course, stmtInsertCourse, course.Visible, course.Creator, course.CustomEMail, course.Description,
		course.EnrollLimitEvents, course.EnrollmentEnd, course.EnrollmentStart, course.ExpirationDate,
		course.Fee, course.OnlyLDAP, course.Speaker, course.Subtitle, course.Title, course.UnsubscribeEnd)
	if err != nil {
		log.Error("failed to insert general course data", "creator ID", course.Creator,
			"title", course.Title, "course", *course, "error", err.Error())
		tx.Rollback()
		return
	}

	if err = course.Events.Insert(tx, &course.ID); err != nil {
		return
	}
	if err = course.CalendarEvents.Insert(tx, &course.ID); err != nil {
		return
	}

	if err = course.Editors.InsertUploaded(tx, &course.ID, TableEditors); err != nil {
		return
	}
	if err = course.Instructors.InsertUploaded(tx, &course.ID, TableInstructors); err != nil {
		return
	}
	if err = course.Blocklist.InsertUploaded(tx, &course.ID, TableBlocklists); err != nil {
		return
	}
	if err = course.Allowlist.InsertUploaded(tx, &course.ID, TableAllowlists); err != nil {
		return
	}

	if err = course.Restrictions.InsertUploaded(tx, course.ID); err != nil {
		return
	}

	tx.Commit()
	return
}

/*InsertFromDraft inserts a new course by duplicating an existing course. */
func (course *Course) InsertFromDraft(v *revel.Validation) (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	authorized := false
	err = tx.Get(&authorized, stmtAuthorizedToEditCourse, course.Creator.Int32, course.ID)
	if err != nil {
		log.Error("failed to retrieve whether the user is authorized or not", "userID",
			course.Creator, "ID", course.ID, "error", err.Error())
		tx.Rollback()
		return
	}

	if !authorized {
		v.ErrorKey("intercept.invalid.action")
		tx.Rollback()
		return
	}

	if err = course.Duplicate(); err != nil {
		return
	}

	tx.Commit()
	return
}

//FeePattern is the regular expression of accepted course fees
var FeePattern = regexp.MustCompile("^([0-9]{1,6}([,|.][0-9]{0,2})?)?")

const (
	stmtGetCourse = `
		SELECT
			id, title, creator, subtitle, visible, active, only_ldap, parent_id,
			description, fee, custom_email, enroll_limit_events, speaker, creation_date,
			enrollment_start, enrollment_end, unsubscribe_end, expiration_date,
			TO_CHAR (creation_date AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') AS creation_date_str,
			TO_CHAR (enrollment_start AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') AS enrollment_start_str,
			TO_CHAR (enrollment_end AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') AS enrollment_end_str,
			TO_CHAR (expiration_date AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') AS expiration_date_str,
			(current_timestamp >= expiration_date) AS expired,

			CASE WHEN unsubscribe_end IS NOT NULL
					THEN TO_CHAR (unsubscribe_end AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI')
				ELSE null
			END AS unsubscribe_end_str

		FROM courses
		WHERE id = $1
	`

	stmtInsertBlankCourse = `
		INSERT INTO courses (title, creator)
		VALUES ($2, $1)
		RETURNING id, title
	`

	stmtCourseIsInactiveOrExpired = `
		SELECT EXISTS (
			SELECT id
			FROM courses
			WHERE id = $1
				AND (
					active = false
					OR (current_timestamp > expiration_date)
				)
		) AS valid
	`

	stmtDuplicateCourse = `
		INSERT INTO courses (
			title, subtitle, creator, custom_email, description, enroll_limit_events, enrollment_end,
			enrollment_start, expiration_date, fee, only_ldap, parent_id, speaker, unsubscribe_end,
			visible
		)
		(
			SELECT
				$2 AS title, subtitle, $3 AS creator, custom_email, description, enroll_limit_events,
				enrollment_end, enrollment_start, expiration_date, fee, only_ldap, parent_id,
				speaker, unsubscribe_end, visible
			FROM courses
			WHERE id = $1
		)
		RETURNING id, title
	`

	stmtInsertCourse = `
		INSERT INTO courses
			(visible, creator, custom_email, description, enroll_limit_events, enrollment_end, enrollment_start,
			expiration_date, fee, only_ldap, speaker, subtitle, title, unsubscribe_end)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
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

	stmtGetCourseVisible = `
		SELECT visible
		FROM courses
		WHERE id = $1
	`

	stmtCourseExpired = `
		SELECT (
			current_timestamp >= expiration_date
			AND active
		) AS expired
		FROM courses
		WHERE id = $1
	`

	stmtIsEditorInstructorOfCourse = `
		SELECT
			EXISTS (
				SELECT true
				FROM editors e
				WHERE e.user_id = $1
					AND e.course_id = $2
			) AS can_edit,
			EXISTS (
				SELECT true
				FROM instructors i
				WHERE i.user_id = $1
					AND i.course_id = $2
			) AS can_manage_participants
	`

	stmtUpdateEnrollmentStatusDueToFee = `
		UPDATE enrolled
		SET status = $2 /* enrolled or awaiting payment */
		WHERE event_id IN (
				SELECT e.id
				FROM events e JOIN courses c ON e.course_id = c.id
				WHERE c.id = $1
			)
			AND status != 1 /* on waitlist */
	`
)
