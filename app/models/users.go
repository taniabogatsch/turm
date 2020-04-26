package models

import (
	"strconv"
	"strings"
	"turm/app"
)

/*Users holds specific users, such as only admins, creators, ... */
type Users []User

/*Get specific users. */
func (users *Users) Get(role Role) (err error) {

	switch role {
	case ADMIN:
		err = app.Db.Select(users, stmtSelectUsers, ADMIN, app.TimeZone)
		if err != nil {
			modelsLog.Error("failed to get users", "role", ADMIN, "error", err.Error())
		}
	case CREATOR:
		err = app.Db.Select(users, stmtSelectUsers, CREATOR, app.TimeZone)
		if err != nil {
			modelsLog.Error("failed to get users", "role", CREATOR, "error", err.Error())
		}
	default:
		modelsLog.Error("invalid role provided", "role", role)
	}
	return
}

/*Search for users. */
func (users *Users) Search(value *string, searchInactive *bool) (err error) {

	//prepare the search value for pattern matching
	values := "%"
	for _, val := range strings.Split(*value, " ") {
		values += val + "%"
	}
	//the value can be the matriculation number
	matrNr, _ := strconv.Atoi(*value) //matrNr is 0 if there is an error

	err = app.Db.Select(users, stmtSearchUsers, values, matrNr, app.TimeZone, *searchInactive)
	if err != nil {
		modelsLog.Error("failed to search users", "values", values,
			"matrNr", matrNr, "error", err.Error())
	}
	return
}

const (
	stmtSearchUsers = `
    SELECT id, lastname, firstname, email, salutation, role, title, academictitle, nameaffix,
      TO_CHAR (lastlogin AT TIME ZONE $3, 'YYYY-MM-DD HH24:MI') as lastlogin
    FROM users
    WHERE (
				(activationcode IS NOT NULL) = $4
				OR activationcode IS NULL
			)
      AND (
        /* all combinations having a nameaffix */
        title || academictitle || firstname || nameaffix || lastname ILIKE $1
        OR title || firstname || nameaffix || lastname ILIKE $1
        OR academictitle || firstname || nameaffix || lastname ILIKE $1
        OR firstname || nameaffix || lastname ILIKE $1

        /* all combinations without a nameaffix */
        OR title || academictitle || firstname || lastname ILIKE $1
        OR title || firstname || lastname ILIKE $1
        OR academictitle || firstname || lastname ILIKE $1
        OR firstname || lastname ILIKE $1

        /* others */
        OR email ILIKE $1
        OR matrnr = $2
      )
    ORDER BY lastname, firstname, id
    LIMIT 5
  `

	stmtSelectUsers = `
		SELECT id, lastname, firstname, email, salutation, title, academictitle, nameaffix,
			TO_CHAR (lastlogin AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') as lastlogin
		FROM users
		WHERE role = $1
			AND activationcode IS NULL
		ORDER BY lastname, firstname, id
	`
)
