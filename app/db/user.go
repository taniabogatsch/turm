package db

import (
	"database/sql"
	"math/rand"
	"strconv"
	"time"
	"turm/app"
	"turm/app/models"

	"github.com/revel/revel"
)

/*Login inserts or updates a user. It returns all session values of that user. */
func Login(user *models.User) (err error) {

	//last login (and first login)
	now := time.Now().Format(revel.TimeFormats[2])

	if !user.Password.Valid { //ldap login

		tx, err := app.Db.Beginx()
		if err != nil {
			revel.AppLog.Error("failed to begin tx in Login()", "error", err.Error())
			return err
		}

		//insert or update users table data
		err = tx.Get(user, stmtLoginLdap,
			user.FirstName, user.LastName, user.EMail, user.Salutation, now, now, user.MatrNr,
			user.AcademicTitle, user.Title, user.NameAffix, user.Affiliations, app.TimeZone)
		if err != nil {
			revel.AppLog.Error("failed to update or insert ldap user", "user", user, "error", err.Error())
			tx.Rollback()
			return err
		}

		if user.MatrNr.Valid && user.FirstLogin == now { //update the courses of study of that user
			revel.AppLog.Debug("first login", "time", user.FirstLogin)
			//TODO: update the courses of study
		}

		tx.Commit()

	} else { //external login

		err = app.Db.Get(user, stmtLoginExtern, now, user.EMail, user.Password)
		if err != nil {
			if err != sql.ErrNoRows {
				revel.AppLog.Error("failed to update external user", "user", user, "error", err.Error())
				return
			}
			err = nil
		}
	}

	return
}

/*Register inserts an external user. It returns all session values of that user. */
func Register(user *models.User) (err error) {

	activationCode := generateActivationCode()

	//last login and first login
	now := time.Now().Format(revel.TimeFormats[2])

	err = app.Db.Get(user, stmtRegisterExtern, user.FirstName, user.LastName, user.EMail,
		user.Salutation, now, now, user.Password, activationCode, user.Language)
	if err != nil {
		revel.AppLog.Error("failed to register external user", "user", user, "error", err.Error())
	}
	user.ActivationCode.String = activationCode
	return
}

/*NewPassword generates a new password for an user. */
func NewPassword(user *models.User) (err error) {

	updatePassword := `
		UPDATE users
		SET password = crypt($1, gen_salt('bf'))
		WHERE email = $2
		RETURNING id, lastname, firstname, email, language
	`
	password := generateActivationCode()

	err = app.Db.Get(user, updatePassword, password, user.EMail)
	if err != nil {
		revel.AppLog.Error("failed to update password", "user", user,
			"password", password, "error", err.Error())
	}
	user.Password.String = password
	return
}

/*VerifyActivationCode verifies an activation code. */
func VerifyActivationCode(activationCode *string, userID *int) (success bool, err error) {

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
		revel.AppLog.Error("failed to begin tx in VerifyActivationCode()", "error", err.Error())
		return
	}

	err = tx.Get(&success, selectCode, *activationCode, *userID)
	if err != nil {
		revel.AppLog.Error("failed to select activation code", "activationCode", *activationCode,
			"userID", *userID, "error", err.Error())
		tx.Rollback()
		return
	}

	if !success {
		return
	}

	_, err = tx.Exec(updateCode, *userID)
	if err != nil {
		revel.AppLog.Error("failed to update activation code", "userID", *userID, "error", err.Error())
		tx.Rollback()
		return
	}

	tx.Commit()
	return
}

/*NewActivationCode creates a new activation code for an user. */
func NewActivationCode(user *models.User) (err error) {

	updateCode := `
		UPDATE users
		SET activationcode = crypt($1, gen_salt('bf'))
		WHERE id = $2
		RETURNING id, lastname, firstname, email, language
	`
	activationCode := generateActivationCode()

	err = app.Db.Get(user, updateCode, activationCode, user.ID)
	if err != nil {
		revel.AppLog.Error("failed to update activation code", "user", user,
			"activationCode", activationCode, "error", err.Error())
	}
	user.ActivationCode.String = activationCode
	return
}

/*SetPrefLanguage sets the preferred language of an user. */
func SetPrefLanguage(userIDSession *string, language *string) (err error) {

	userID, err := strconv.Atoi(*userIDSession)
	if err != nil {
		revel.AppLog.Error("failed to parse userID from userIDSession",
			"userIDSession", *userIDSession, "error", err.Error())
		return
	}

	updateLanguage := `UPDATE users SET language = $1 WHERE id = $2`
	_, err = app.Db.Exec(updateLanguage, *language, userID)
	if err != nil {
		revel.AppLog.Error("failed to update language", "userID", userID, "error", err.Error())
	}
	return
}

//generateActivationCode generates an activation code.
func generateActivationCode() string {

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
	return string(b)
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
		RETURNING id, lastname, firstname, email, role, language
	`
)
