package models

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"math/rand"
	"strconv"
	"strings"
	"time"
	"turm/app"

	"github.com/jmoiron/sqlx"
	"github.com/revel/revel"
)

/*Salutation is a type for encoding different forms of address. */
type Salutation int

const (
	//NONE is for no form of address
	NONE Salutation = iota
	//MR is for Mr.
	MR
	//MS is for Ms.
	MS
)

func (s Salutation) String() string {
	return [...]string{"none", "mr", "ms"}[s]
}

/*Role is a type for encoding different user roles. */
type Role int

const (
	//USER is the default role without any extra privileges
	USER Role = iota
	//CREATOR allows the creation of courses
	CREATOR
	//ADMIN grants all privileges
	ADMIN
)

func (u Role) String() string {
	return [...]string{"user", "creator", "admin"}[u]
}

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
}

/*Validate User fields of newly registered users. */
func (user *User) Validate(v *revel.Validation) {

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

/*Credentials entered at the login page. */
type Credentials struct {
	Username     string
	EMail        string
	Password     string
	StayLoggedIn bool
}

/*Validate the credentials. */
func (credentials *Credentials) Validate(v *revel.Validation) {

	if credentials.Username != "" { //ldap login credentials

		v.MaxSize(credentials.Username, 255).
			MessageKey("validation.invalid.username")

		v.Check(credentials.EMail,
			NotRequired{},
		).MessageKey("validation.invalid.credentials")

	} else if credentials.EMail != "" { //external login credentials

		v.Required(credentials.EMail).
			MessageKey("validation.invalid.email")

		v.Email(credentials.EMail).
			MessageKey("validation.invalid.email")

		v.Check(credentials.Username,
			NotRequired{},
		).MessageKey("validation.invalid.credentials")

	} else { //neither username nor e-mail address was provided
		v.ErrorKey("validation.invalid.username")
	}

	v.Check(credentials.Password,
		revel.Required{},
		revel.MaxSize{127},
	).MessageKey("validation.invalid.password")
}

/*Studies holds all courses of study of an user. */
type Studies []Study

/*Study is a model of the studies table. */
type Study struct {
	UserID            int    `db:"user_id, primarykey"`
	Semester          int    `db:"semester"`
	DegreeID          int    `db:"degree_id, primarykey"`
	CourseOfStudiesID int    `db:"course_of_studies_id, primarykey"`
	Degree            string `db:"degree"`            //not a field in the studies table
	CourseOfStudies   string `db:"course_of_studies"` //not a field in the studies table
}

/*Validate Studies fields when loaded from the user enrollment file. */
func (studies *Study) Validate(v *revel.Validation) {
	//TODO
}

/*Select all courses of studies of a user. */
func (studies *Studies) Select(tx *sqlx.Tx, userID *int) (err error) {

	err = tx.Select(studies, stmtSelectUserCoursesOfStudies, *userID)
	if err != nil {
		log.Error("failed to get courses of studies of user", "userID", *userID,
			"error", err.Error())
		tx.Rollback()
	}
	return
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
func (user *User) Register() (err error) {

	activationCode := generateCode()

	//last login and first login
	now := time.Now().Format(revel.TimeFormats[0])

	err = app.Db.Get(user, stmtRegisterExtern, user.FirstName, user.LastName, user.EMail,
		user.Salutation, now, now, user.Password, activationCode, user.Language)
	if err != nil {
		log.Error("failed to register external user", "user", user, "error", err.Error())
	}
	user.ActivationCode.String = activationCode
	return
}

/*NewPassword generates a new password for an user. */
func (user *User) NewPassword() (err error) {

	password := generateCode()

	err = app.Db.Get(user, stmtUpdatePassword, password, user.EMail)
	if err != nil {
		log.Error("failed to update password", "user", user,
			"password", password, "error", err.Error())
	}
	user.Password.String = password
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
func (user *User) SetPrefLanguage(userIDSession *string) (err error) {

	user.ID, err = strconv.Atoi(*userIDSession)
	if err != nil {
		log.Error("failed to parse userID from session",
			"userIDSession", *userIDSession, "error", err.Error())
		return
	}

	err = app.Db.Get(user, stmtUpdateLanguage, user.Language.String, user.ID)
	if err != nil {
		log.Error("failed to update language", "userID", user.ID,
			"language", user.Language, "error", err.Error())
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
func (user *User) HasElevatedRights(ID *int) (authorized, expired bool, err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
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

//generateCode generates an activation code or a random password.
func generateCode() string {

	//to create a unique random, we need to take the time in nanoseconds as seed
	rand.Seed(time.Now().UTC().UnixNano())
	//characters that can be used in the activation code (no l, I, L, O, 0, 1)
	var characters = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ23456789"
	//the length of the activation code
	b := make([]byte, 7)

	//generate the code
	for i := range b {
		b[i] = characters[rand.Intn(len(characters))]
	}

	log.Debug("generated code", "code", string(b))
	return string(b)
}

/* --- CUSTOM SQL TYPES --- */

/*Affiliations contains all affiliations of a user. */
type Affiliations []string

/*NullAffiliations represents affiliations that may be null. */
type NullAffiliations struct {
	Affiliations Affiliations
	Valid        bool //Valid is true if Affiliations is not NULL
}

/*Value constructs a SQL Value from NullAffiliations. */
func (affiliations NullAffiliations) Value() (driver.Value, error) {

	if !affiliations.Valid {
		return nil, nil
	}

	var str string
	for _, affiliation := range affiliations.Affiliations {
		str += `"` + affiliation + `",`
	}
	return driver.Value("{" + strings.TrimRight(str, ",") + "}"), nil
}

/*Scan constructs NullAffiliations from a SQL Value. */
func (affiliations *NullAffiliations) Scan(value interface{}) error {

	if value == nil {
		affiliations.Affiliations = []string{""}
		affiliations.Valid = false
		return nil
	}

	affiliations.Valid = true

	switch value.(type) {
	case string:
		str := value.(string)
		str = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(str, "{", ""), "}", ""))
		affiliations.Affiliations = strings.Split(str, ",")
	default:
		return errors.New("incompatible type for Affiliations")
	}
	return nil
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

	stmtUpdatePassword = `
		UPDATE users
		SET password = CRYPT($1, gen_salt('bf'))
		WHERE email = $2
		RETURNING
			/* data to send notification e-mail containing the new password */
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
)
