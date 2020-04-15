package database

import (
	"math/rand"
	"time"
	"turm/app"
	"turm/app/models"

	"github.com/revel/revel"
)

/*Login inserts or updates a user. It returns all session values of that user. */
func Login(user *models.User) (err error) {

	//last login (and first login)
	now := time.Now().Format("2006-01-02 15:04:05")

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
			revel.AppLog.Error("failed to update external user", "user", user, "error", err.Error())
			return
		}
	}

	return
}

/*Register inserts an external user. It returns all session values of that user. */
func Register(user *models.User) (err error) {

	user.ActivationCode.String = generateActivationCode()
	user.ActivationCode.Valid = true
	activationCode := user.ActivationCode.String

	//last login and first login
	now := time.Now().Format("2006-01-02 15:04:05")

	err = app.Db.Get(user, stmtRegisterExtern, user.FirstName, user.LastName, user.EMail,
		user.Salutation, now, now, user.Password, user.ActivationCode)
	if err != nil {
		revel.AppLog.Error("failed to register external user", "user", user, "error", err.Error())
	}
	user.ActivationCode.String = activationCode
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
		RETURNING id, lastname, firstname, email, role, matrnr,
      TO_CHAR (firstlogin AT TIME ZONE $12, 'YYYY-MM-DD HH24:MI:SS') as firstlogin
	`

	stmtLoginExtern = `
    UPDATE users
    SET lastlogin = $1
    WHERE email = $2
      AND password = crypt($3, password)
    RETURNING id, lastname, firstname, email, role, activationcode
  `

	stmtRegisterExtern = `
		INSERT INTO users (
			firstname, lastname, email, salutation, role, lastlogin,
			firstlogin, password, activationcode
		)
		VALUES ($1, $2, $3, $4, 0, $5, $6, crypt($7, gen_salt('bf')), crypt($8, gen_salt('bf')))
		RETURNING id, lastname, firstname, email, role
	`
)
