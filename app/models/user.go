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
	LastName   string         `db:"lastname"`
	FirstName  string         `db:"firstname"`
	EMail      string         `db:"email, unique"`
	Salutation Salutation     `db:"salutation"`
	Role       Role           `db:"role"`
	LastLogin  string         `db:"lastlogin"`
	FirstLogin string         `db:"firstlogin"`
	Language   sql.NullString `db:"language"`

	//ldap user fields
	MatrNr        sql.NullInt32    `db:"matrnr, unique"`
	AcademicTitle sql.NullString   `db:"academictitle"`
	Title         sql.NullString   `db:"title"`
	NameAffix     sql.NullString   `db:"nameaffix"`
	Affiliations  NullAffiliations `db:"affiliations"`
	Studies       []Studies        ``

	//external user fields
	Password       sql.NullString `db:"password"`
	PasswordRepeat string         `` //not a field in the respective table
	ActivationCode sql.NullString `db:"activationcode"`
}

/*Validate user fields of newly registered users. */
func (user *User) Validate(v *revel.Validation) {

	user.EMail = strings.ToLower(user.EMail)

	v.Check(user.LastName,
		revel.Required{},
		revel.MaxSize{255},
	).MessageKey("validation.invalid.lastname")

	v.Check(user.FirstName,
		revel.Required{},
		revel.MaxSize{255},
	).MessageKey("validation.invalid.firstname")

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

	isLdapEMail := !strings.Contains(user.EMail, app.EMailSuffix)
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

	if user.Salutation != NONE && user.Salutation != MR && user.Salutation != MS {
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

/*Get all data of an user. */
func (user *User) Get(tx *sqlx.Tx) (err error) {

	err = tx.Get(user, stmtGetUser, app.TimeZone, user.ID)
	if err != nil {
		modelsLog.Error("failed to get user", "user", user, "error", err.Error())
		tx.Rollback()
	}

	//TODO: get courses of studies

	return
}

/*Login inserts or updates a user. It provides all session values of that user. */
func (user *User) Login() (err error) {

	//last login (and first login)
	now := time.Now().Format(revel.TimeFormats[0])

	if !user.Password.Valid { //ldap login

		modelsLog.Debug("ldap login")

		tx, err := app.Db.Beginx()
		if err != nil {
			modelsLog.Error("failed to begin tx", "error", err.Error())
			return err
		}

		//insert or update users table data
		err = tx.Get(user, stmtLoginLdap,
			user.FirstName, user.LastName, user.EMail, user.Salutation, now, now, user.MatrNr,
			user.AcademicTitle, user.Title, user.NameAffix, user.Affiliations, app.TimeZone)
		if err != nil {
			modelsLog.Error("failed to update or insert ldap user", "user", user, "error", err.Error())
			tx.Rollback()
			return err
		}

		if user.MatrNr.Valid && user.FirstLogin == now { //update the courses of study of that user
			modelsLog.Debug("first login", "time", user.FirstLogin)
			//TODO: update the courses of study
		}

		tx.Commit()

	} else { //external login

		modelsLog.Debug("external login")

		err = app.Db.Get(user, stmtLoginExtern, now, user.EMail, user.Password)
		if err != nil {
			if err != sql.ErrNoRows {
				modelsLog.Error("failed to update external user", "user", user, "error", err.Error())
				return
			}
			err = nil
		}
	}
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
		modelsLog.Error("failed to register external user", "user", user, "error", err.Error())
	}
	user.ActivationCode.String = activationCode
	return
}

/*NewPassword generates a new password for an user. */
func (user *User) NewPassword() (err error) {

	updatePassword := `
		UPDATE users
		SET password = crypt($1, gen_salt('bf'))
		WHERE email = $2
		RETURNING
			/* data to send notification e-mail containing the new password */
			id, lastname, firstname, email, language, salutation
	`
	password := generateCode()

	err = app.Db.Get(user, updatePassword, password, user.EMail)
	if err != nil {
		modelsLog.Error("failed to update password", "user", user,
			"password", password, "error", err.Error())
	}
	user.Password.String = password
	return
}

/*VerifyActivationCode verifies an activation code. */
func (user *User) VerifyActivationCode() (success bool, err error) {

	selectCode := `
		SELECT EXISTS (
			SELECT true
			FROM users
			WHERE id = $2
				AND (
					activationcode = CRYPT($1, activationcode)
					OR
					activationcode IS NULL
				)
		) AS success
	`
	updateCode := `UPDATE users SET activationcode = null WHERE id = $1`

	tx, err := app.Db.Beginx()
	if err != nil {
		modelsLog.Error("failed to begin tx", "error", err.Error())
		return
	}

	err = tx.Get(&success, selectCode, user.ActivationCode.String, user.ID)
	if err != nil {
		modelsLog.Error("failed to select activation code", "user", user, "error", err.Error())
		tx.Rollback()
		return
	}

	if !success {
		modelsLog.Debug("invalid activation code, verification failed",
			"user", user)
		tx.Commit()
		return
	}

	_, err = tx.Exec(updateCode, user.ID)
	if err != nil {
		modelsLog.Error("failed to update activation code", "user", user, "error", err.Error())
		tx.Rollback()
		return
	}

	tx.Commit()
	return
}

/*NewActivationCode creates a new activation code for an user. */
func (user *User) NewActivationCode() (err error) {

	updateCode := `
		UPDATE users
		SET activationcode = crypt($1, gen_salt('bf'))
		WHERE id = $2
		RETURNING
			/* data to send notification e-mail containing the new code */
			id, lastname, firstname, email, language, salutation
	`
	activationCode := generateCode()

	err = app.Db.Get(user, updateCode, activationCode, user.ID)
	if err != nil {
		modelsLog.Error("failed to update activation code", "user", user,
			"activationCode", activationCode, "error", err.Error())
	}
	user.ActivationCode.String = activationCode
	return
}

/*SetPrefLanguage sets the preferred language of an user. */
func (user *User) SetPrefLanguage(userIDSession *string) (err error) {

	updateLanguage := `
		UPDATE users
		SET language = $1
		WHERE id = $2
		RETURNING id
	`

	user.ID, err = strconv.Atoi(*userIDSession)
	if err != nil {
		modelsLog.Error("failed to parse userID from session",
			"userIDSession", *userIDSession, "error", err.Error())
		return
	}

	err = app.Db.Get(updateLanguage, user.Language.String, user.ID)
	if err != nil {
		modelsLog.Error("failed to update language", "userID", user.ID,
			"language", user.Language, "error", err.Error())
	}
	return
}

/*ChangeRole of an user. */
func (user *User) ChangeRole() (err error) {

	updateRole := `
		UPDATE users
		SET role = $1
		WHERE id = $2
		RETURNING
			/* data to send notification e-mail about the new role */
			id, firstname, lastname, role, language,
			academictitle, email, nameaffix, salutation, title
	`

	err = app.Db.Get(user, updateRole, user.Role, user.ID)
	if err != nil {
		modelsLog.Error("failed to update user role", "userID", user.ID,
			"role", user.Role, "error", err.Error())
	}
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

	modelsLog.Debug("generated code", "code", string(b))
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
			firstname, lastname, email, salutation, role, lastlogin,
			firstlogin, matrnr, academictitle, title, nameaffix, affiliations
		)
		VALUES ($1, $2, $3, $4, 0, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (email)
		DO UPDATE
			SET
				firstname = $1, lastname = $2, salutation = $4, lastlogin = $5,
				matrnr = $7, academictitle = $8, title = $9, nameaffix = $10, affiliations = $11
		RETURNING id, lastname, firstname, email, role, matrnr, language,
			TO_CHAR (firstlogin AT TIME ZONE $12, 'YYYY-MM-DD HH24:MI:SS') as firstlogin
	`

	stmtLoginExtern = `
		UPDATE users
		SET lastlogin = $1
		WHERE email = $2
			AND password = crypt($3, password)
		RETURNING id, lastname, firstname, email, role, activationcode, language
	`

	stmtRegisterExtern = `
		INSERT INTO users (
			firstname, lastname, email, salutation, role, lastlogin,
			firstlogin, password, activationcode, language
		)
		VALUES ($1, $2, $3, $4, 0, $5, $6, crypt($7, gen_salt('bf')), crypt($8, gen_salt('bf')), $9)
		RETURNING
			/* data to send notification e-mail containing the activation */
			id, lastname, firstname, email, role, language, salutation
	`

	stmtGetUser = `
		SELECT
			id, lastname, firstname, email, salutation, role, activationcode,
			language, matrnr, academictitle, title, nameaffix, affiliations,
			TO_CHAR (lastlogin AT TIME ZONE $1, 'YYYY-MM-DD HH24:MI') as lastlogin,
			TO_CHAR (firstlogin AT TIME ZONE $1, 'YYYY-MM-DD HH24:MI') as firstlogin
		FROM users
		WHERE id = $2
	`
)
