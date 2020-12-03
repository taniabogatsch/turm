package models

import (
	"database/sql"
	"strings"
	"time"
	"turm/app"

	"github.com/jmoiron/sqlx"
	"github.com/revel/revel"
)

/*User is a model of the users table. */
type User struct {
	ID         int            `db:"id, primarykey, autoincrement"`
	LastName   string         `db:"last_name"`
	FirstName  string         `db:"first_name"`
	EMail      string         `db:"email, unique"`
	Salutation Salutation     `db:"salutation"`
	Role       Role           `db:"role"`
	LastLogin  string         `db:"last_login"`
	FirstLogin string         `db:"first_login"`
	Language   sql.NullString `db:"language"`

	//ldap user fields
	MatrNr        sql.NullInt32    `db:"matr_nr, unique"`
	AcademicTitle sql.NullString   `db:"academic_title"`
	Title         sql.NullString   `db:"title"`
	NameAffix     sql.NullString   `db:"name_affix"`
	Affiliations  NullAffiliations `db:"affiliations"`
	Studies       Studies          ``

	//external user fields
	Password       sql.NullString `db:"password"`
	PasswordRepeat string         `` //not a field in the respective table
	ActivationCode sql.NullString `db:"activation_code"`

	//not a field in the resprective table
	IsEditor     bool
	IsInstructor bool

	//used for event enrollment
	IsLDAP bool `db:"is_ldap"`

	//used for profile page
	ActiveEnrollments  Enrollments
	ExpiredEnrollments Enrollments
}

/*Validate User fields of newly registered users. */
func (user *User) Validate(tx *sqlx.Tx, v *revel.Validation) {

	user.EMail = strings.ToLower(user.EMail)

	user.LastName = strings.TrimSpace(user.LastName)
	v.Check(user.LastName,
		revel.Required{},
		revel.MaxSize{255},
	).MessageKey("validation.invalid.lastname")

	user.FirstName = strings.TrimSpace(user.FirstName)
	v.Check(user.FirstName,
		revel.Required{},
		revel.MaxSize{255},
	).MessageKey("validation.invalid.firstname")

	user.EMail = strings.TrimSpace(user.EMail)
	v.Check(user.EMail,
		revel.Required{},
		revel.MaxSize{255},
	).MessageKey("validation.invalid.email")

	v.Email(user.EMail).
		MessageKey("validation.invalid.email")

	data := ValidateUniqueData{
		Column: "email",
		Table:  "users",
		Value:  user.EMail,
		Tx:     tx,
	}
	v.Check(data,
		Unique{},
	).MessageKey("validation.email.notUnique")

	isLdapEMail := !strings.Contains(user.EMail, app.Mailer.Suffix)
	v.Required(isLdapEMail).
		MessageKey("validation.email.ldap")

	v.Check(user.Password.String,
		revel.Required{},
		revel.MaxSize{127},
		revel.MinSize{6},
	).MessageKey("validation.invalid.passwords")

	equal := (user.Password.String == user.PasswordRepeat)
	v.Required(equal).
		MessageKey("validation.invalid.passwords")
	v.Required(user.PasswordRepeat).
		MessageKey("validation.invalid.passwords")

	user.Password.Valid = true

	v.Check(user.Language.String,
		revel.Required{},
		LanguageValidator{},
	).MessageKey("validation.invalid.language")

	user.Language.Valid = true

	if user.Salutation < NONE || user.Salutation > MS {
		v.ErrorKey("validation.invalid.salutation")
	}
}

/*Get all data of an user. */
func (user *User) Get(tx *sqlx.Tx) (err error) {

	err = tx.Get(user, stmtGetUser, app.TimeZone, user.ID)
	if err != nil {
		log.Error("failed to get user", "user", user, "error", err.Error())
		tx.Rollback()
	}

	if user.IsLDAP {
		err = user.Studies.Select(tx, &user.ID)
	}

	return
}

/*GetBasicData returns basic information of an user. */
func (user *User) GetBasicData(tx *sqlx.Tx) (err error) {

	err = tx.Get(user, stmtSelectUser, user.ID)
	if err != nil {
		log.Error("failed to get user", "user", user, "error", err.Error())
		tx.Rollback()
	}
	return
}

/*GetProfileData returns all profile information of the user. */
func (user *User) GetProfileData() (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return err
	}

	//get user data
	if err = user.Get(tx); err != nil {
		return
	}

	//get all active enrollments
	err = user.ActiveEnrollments.SelectByUser(tx, &user.ID, false)
	if err != nil {
		return
	}
	//get all expired enrollments
	err = user.ExpiredEnrollments.SelectByUser(tx, &user.ID, true)
	if err != nil {
		return
	}

	//TODO: get all calendar slots

	tx.Commit()
	return
}

/*Login inserts or updates a user. It provides all session values of that user. */
func (user *User) Login() (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return err
	}

	//last login (and first login)
	now := time.Now().Format(revel.TimeFormats[0])

	if !user.Password.Valid { //ldap login

		log.Debug("ldap login")

		//insert or update users table data
		err = tx.Get(user, stmtLoginLdap,
			user.FirstName, user.LastName, user.EMail, user.Salutation, now, now, user.MatrNr,
			user.AcademicTitle, user.Title, user.NameAffix, user.Affiliations, app.TimeZone)
		if err != nil {
			log.Error("failed to update or insert ldap user", "user", user, "error", err.Error())
			tx.Rollback()
			return
		}

		user.IsEditor, user.IsInstructor, err = user.IsEditorInstructor(tx)
		if err != nil {
			return
		}

		if user.MatrNr.Valid && user.FirstLogin == now { //update the courses of study of that user
			log.Debug("first login", "time", user.FirstLogin)
			//TODO: update the courses of study
		}

	} else { //external login

		log.Debug("external login")

		err = tx.Get(user, stmtLoginExtern, now, user.EMail, user.Password)
		if err != nil {
			if err != sql.ErrNoRows {
				log.Error("failed to update external user", "user", user, "error", err.Error())
				tx.Rollback()
				return
			}
			err = nil
		}

		user.IsEditor, user.IsInstructor, err = user.IsEditorInstructor(tx)
		if err != nil {
			return
		}
	}

	tx.Commit()
	return
}

/*Register inserts an external user. It provides all session values of that user. */
func (user *User) Register(v *revel.Validation) (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	if user.Validate(tx, v); v.HasErrors() {
		tx.Rollback()
		return
	}

	activationCode := generateCode()

	//last login and first login
	now := time.Now().Format(revel.TimeFormats[0])

	err = tx.Get(user, stmtRegisterExtern, user.FirstName, user.LastName, user.EMail,
		user.Salutation, now, now, user.Password, activationCode, user.Language)
	if err != nil {
		log.Error("failed to register external user", "user", user, "error", err.Error())
		tx.Rollback()
		return
	}

	user.ActivationCode.String = activationCode
	tx.Commit()
	return
}

/*GenerateNewPassword for an user. */
func (user *User) GenerateNewPassword(v *revel.Validation) (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	v.Check(user.EMail,
		revel.Required{},
		revel.MaxSize{255},
	).MessageKey("validation.invalid.email")
	v.Email(user.EMail).
		MessageKey("validation.invalid.email")

	isLdapEMail := !strings.Contains(user.EMail, app.Mailer.Suffix)
	v.Required(isLdapEMail).
		MessageKey("validation.email.ldap")

	data := ValidateUniqueData{
		Column: "email",
		Table:  "users",
		Value:  user.EMail,
		Tx:     tx,
	}
	v.Check(data,
		NotUnique{},
	).MessageKey("validation.invalid.email")

	if v.HasErrors() {
		tx.Rollback()
		return
	}

	password := generateCode()

	err = tx.Get(user, stmtUpdatePasswordByEMail, password, user.EMail)
	if err != nil {
		log.Error("failed to update password", "user", *user,
			"password", password, "error", err.Error())
		tx.Rollback()
		return
	}

	user.Password.String = password
	tx.Commit()
	return
}

/*NewPassword sets a new password for an user. */
func (user *User) NewPassword(newPw1, newPw2 string, v *revel.Validation) (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return err
	}

	if newPw1 != newPw2 {
		v.ErrorKey("validation.invalid.passwords")
		tx.Rollback()
		return
	}

	//ensure that the old password is valid
	match := false
	err = tx.Get(&match, stmtPasswordsMatch, user.ID, user.Password)
	if err != nil {
		log.Error("failed to validate if passwords match", "userID",
			user.ID, "password", user.Password, "error", err.Error())
		tx.Rollback()
		return
	} else if !match {
		v.ErrorKey("validation.invalid.password.match")
		tx.Rollback()
		return
	}

	err = tx.Get(user, stmtUpdatePassword, newPw1, user.ID)
	if err != nil {
		log.Error("failed to update password", "user", user,
			"newPw1", newPw1, "error", err.Error())
		tx.Rollback()
		return
	}

	tx.Commit()
	return
}

/*VerifyActivationCode verifies an activation code. */
func (user *User) VerifyActivationCode() (success bool, err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	err = tx.Get(&success, stmtSelectCode, user.ActivationCode.String, user.ID)
	if err != nil {
		log.Error("failed to select activation code", "user", user, "error", err.Error())
		tx.Rollback()
		return
	}

	if !success {
		log.Debug("invalid activation code, verification failed",
			"user", user)
		tx.Commit()
		return
	}

	_, err = tx.Exec(stmtUpdateCode, user.ID)
	if err != nil {
		log.Error("failed to update activation code", "user", user, "error", err.Error())
		tx.Rollback()
		return
	}

	tx.Commit()
	return
}

/*NewActivationCode creates a new activation code for an user. */
func (user *User) NewActivationCode() (err error) {

	activationCode := generateCode()

	err = app.Db.Get(user, stmtUpdateCodeReturningData, activationCode, user.ID)
	if err != nil {
		log.Error("failed to update activation code", "user", user,
			"activationCode", activationCode, "error", err.Error())
	}
	user.ActivationCode.String = activationCode
	return
}

/*SetPrefLanguage sets the preferred language of an user. */
func (user *User) SetPrefLanguage() (err error) {

	err = app.Db.Get(user, stmtUpdateLanguage, user.Language, user.ID)
	if err != nil {
		log.Error("failed to update language", "user", *user,
			"error", err.Error())
	}
	return
}

/*ChangeRole of an user. */
func (user *User) ChangeRole() (err error) {

	err = app.Db.Get(user, stmtUpdateRole, user.Role, user.ID)
	if err != nil {
		log.Error("failed to update user role", "userID", user.ID,
			"role", user.Role, "error", err.Error())
	}
	return
}

/*IsEditorInstructor returns whether a user is an editor or instructor or not. */
func (user *User) IsEditorInstructor(tx *sqlx.Tx) (bool, bool, error) {

	type bools struct {
		IsEditor     bool `db:"is_editor"`
		IsInstructor bool `db:"is_instructor"`
	}
	var data bools

	err := tx.Get(&data, stmtIsEditorInstructor, user.ID)
	if err != nil {
		log.Error("failed to get isEditor and isInstructor", "userID",
			user.ID, "error", err.Error())
		tx.Rollback()
	}
	return data.IsEditor, data.IsInstructor, err
}

/*AuthorizedToEdit returns whether a user is authorized to edit a course or not. */
func (user *User) AuthorizedToEdit(table *string, ID *int) (authorized, expired bool, err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	switch *table {
	case "calendar_events":
		err = tx.Get(ID, stmtGetCourseIDByCalendarEvent, *ID)
	case "events":
		err = tx.Get(ID, stmtGetCourseIDByEvent, *ID)
	case "meetings":
		err = tx.Get(ID, stmtGetCourseIDByMeeting, *ID)
	}

	if err != nil {
		log.Error("failed to get course ID", "ID", *ID, "error", err.Error())
		tx.Rollback()
		return
	}

	err = tx.Get(&expired, stmtCourseExpired, *ID)
	if err != nil {
		log.Error("failed to get whether the course expired or not", "ID", *ID,
			"error", err.Error())
		tx.Rollback()
		return
	}

	if user.ID != 0 {

		switch *table {
		case "courses", "events", "calendar_events", "meetings":
			err = tx.Get(&authorized, stmtAuthorizedToEditCourse, user.ID, *ID)
		case "onlyCreator":
			err = tx.Get(&authorized, stmtIsCreator, user.ID, *ID)
		}

		if err != nil {
			log.Error("failed to retrieve whether the user is authorized or not", "userID", user.ID,
				"ID", *ID, "error", err.Error())
			tx.Rollback()
			return
		}
	}

	tx.Commit()
	return
}

/*HasElevatedRights returns whether a user is an instructor, editor, creator or
admin (of a course). */
func (user *User) HasElevatedRights(ID *int, table string) (authorized, expired bool, err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	if table == "events" {
		err = tx.Get(ID, stmtGetCourseIDByEvent, *ID)
		if err != nil {
			log.Error("failed to get course ID by event ID", "ID", *ID, "error", err.Error())
			tx.Rollback()
			return
		}
	}

	err = tx.Get(&expired, stmtCourseExpired, *ID)
	if err != nil {
		log.Error("failed to get whether the course expired or not", "userID", user.ID,
			"ID", *ID, "error", err.Error())
		tx.Rollback()
		return
	}

	if user.ID != 0 {
		err = tx.Get(&authorized, stmtAuthorizedToManageParticipants, user.ID, *ID)
		if err != nil {
			log.Error("failed to get whether the user has elevated rights or not", "userID", user.ID,
				"ID", *ID, "error", err.Error())
			tx.Rollback()
			return
		}
	}

	tx.Commit()
	return
}

const (
	stmtLoginLdap = `
		INSERT INTO users (
			first_name, last_name, email, salutation, role, last_login,
			first_login, matr_nr, academic_title, title, name_affix, affiliations
		)
		VALUES ($1, $2, $3, $4, 0, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (email)
		DO UPDATE
			SET
				first_name = $1, last_name = $2, salutation = $4, last_login = $5,
				matr_nr = $7, academic_title = $8, title = $9, name_affix = $10, affiliations = $11
		RETURNING id, last_name, first_name, email, role, matr_nr, language,
			TO_CHAR (first_login AT TIME ZONE $12, 'YYYY-MM-DD HH24:MI:SS') as first_login
	`

	stmtLoginExtern = `
		UPDATE users
		SET last_login = $1
		WHERE email = $2
			AND password = CRYPT($3, password)
		RETURNING id, last_name, first_name, email, role, activation_code, language
	`

	stmtRegisterExtern = `
		INSERT INTO users (
			first_name, last_name, email, salutation, role, last_login,
			first_login, password, activation_code, language
		)
		VALUES ($1, $2, $3, $4, 0, $5, $6, CRYPT($7, gen_salt('bf')), CRYPT($8, gen_salt('bf')), $9)
		RETURNING
			/* data to send notification e-mail containing the activation */
			id, last_name, first_name, email, role, language, salutation
	`

	stmtGetUser = `
		SELECT
			id, last_name, first_name, email, salutation, role, activation_code,
			language, matr_nr, academic_title, title, name_affix, affiliations,
			TO_CHAR (last_login AT TIME ZONE $1, 'YYYY-MM-DD HH24:MI') as last_login,
			TO_CHAR (first_login AT TIME ZONE $1, 'YYYY-MM-DD HH24:MI') as first_login,
			(password IS NULL) AS is_ldap
		FROM users
		WHERE id = $2
	`

	stmtSelectUser = `
		SELECT id, email, first_name, last_name, salutation, title, academic_title, name_affix
		FROM users WHERE id = $1
	`

	stmtUpdatePasswordByEMail = `
		UPDATE users
		SET password = CRYPT($1, gen_salt('bf'))
		WHERE email = $2
		RETURNING
			/* data to send notification e-mail containing the new password */
			id, last_name, first_name, email, language, salutation
	`

	stmtUpdatePassword = `
		UPDATE users
		SET password = CRYPT($1, gen_salt('bf'))
		WHERE id = $2
		RETURNING
			/* data to send notification e-mail containing */
			id, last_name, first_name, email, language, salutation
	`

	stmtSelectCode = `
		SELECT EXISTS (
			SELECT true
			FROM users
			WHERE id = $2
				AND (
					activation_code = CRYPT($1, activation_code)
					OR
					activation_code IS NULL
				)
		) AS success
	`

	stmtUpdateCode = `
		UPDATE users
		SET activation_code = NULL
		WHERE id = $1
	`

	stmtUpdateCodeReturningData = `
		UPDATE users
		SET activation_code = CRYPT($1, gen_salt('bf'))
		WHERE id = $2
		RETURNING
			/* data to send notification e-mail containing the new code */
			id, last_name, first_name, email, language, salutation
	`

	stmtUpdateLanguage = `
		UPDATE users
		SET language = $1
		WHERE id = $2
		RETURNING id, language
	`

	stmtUpdateRole = `
		UPDATE users
		SET role = $1
		WHERE id = $2
		RETURNING
			/* data to send notification e-mail about the new role */
			id, first_name, last_name, role, language,
			academic_title, email, name_affix, salutation, title
	`

	stmtAuthorizedToEditCourse = `
		SELECT EXISTS (
			SELECT true
			FROM courses
			WHERE id = $2
				AND creator = $1

			UNION

			SELECT true
			FROM editors
			WHERE user_id = $1
				AND course_id = $2

		) AS authorized
	`

	stmtIsEditorInstructor = `
		SELECT
			EXISTS (
				SELECT true
				FROM editors e
				WHERE e.user_id = $1
			) AS is_editor,
			EXISTS (
				SELECT true
				FROM instructors i
				WHERE i.user_id = $1
			) AS is_instructor
	`

	stmtIsCreator = `
		SELECT true AS authorized
		FROM courses
		WHERE id = $2
			AND creator = $1
	`

	stmtSelectUserCoursesOfStudies = `
		SELECT user_id, semester, degree_id,
			course_of_studies_id, d.name AS degree,
			c.name AS course_of_studies
		FROM studies s LEFT OUTER JOIN
			degrees d ON s.degree_id = d.id LEFT OUTER JOIN
			courses_of_studies c ON s.course_of_studies_id = c.id
		WHERE user_id = $1
	`

	stmtAuthorizedToManageParticipants = `
		SELECT EXISTS (
			SELECT true
			FROM courses
			WHERE id = $2
				AND creator = $1

			UNION

			SELECT true
			FROM editors
			WHERE user_id = $1
				AND course_id = $2

			UNION

			SELECT true
			FROM instructors
			WHERE user_id = $1
				AND course_id = $2

		) AS authorized
	`

	stmtPasswordsMatch = `
		SELECT EXISTS (
			SELECT id
			FROM users
			WHERE id = $1
				AND password = CRYPT($2, password)
		) AS match
	`
)
