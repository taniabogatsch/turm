/*Package models contains all database tables as structs and their validation
functions. It also contains additional structs representing front end data, such
as the user login credentials.*/
package models

import (
	"math/rand"
	"time"
	"turm/app"

	"github.com/jmoiron/sqlx"
	"github.com/revel/revel"
)

var (
	//log all model errors
	log = revel.AppLog.New("section", "models")
)

/*BelongsToElement returns whether e.g. an event belongs to a course. */
func BelongsToElement(table, column1, column2 string, ID1, ID2 int) (belongs bool, err error) {

	stmt := `
	SELECT EXISTS (
		SELECT id
		FROM ` + table + `
		WHERE ` + column1 + ` = $1
			AND ` + column2 + ` = $2
	) AS belongs
`

	if table == "calendar_exceptions" || table == "day_templates" {

		stmt = `
		SELECT EXISTS (
			SELECT t.id
			FROM ` + table + ` t JOIN calendar_events e ON t.calendar_event_id = e.id
			WHERE e.` + column1 + ` = $1
				AND t.` + column2 + ` = $2
		) AS belongs
	`
	}

	err = app.Db.Get(&belongs, stmt, ID1, ID2)
	if err != nil {
		log.Error("failed to validate if an element belongs to another element", "stmt",
			stmt, "ID1", ID1, "ID2", ID2, "error", err.Error())
	}

	return
}

//deleteByID deletes an entry from a table by its ID
func deleteByID(column, table string, value interface{}, tx *sqlx.Tx) (err error) {

	stmt := `DELETE FROM ` + table + ` WHERE ` + column + ` = $1`

	if tx == nil {
		_, err = app.Db.Exec(stmt, value)
	} else {
		_, err = tx.Exec(stmt, value)
		if err != nil {
			tx.Rollback()
		}
	}
	if err != nil {
		log.Error("cannot delete entry", "column", column, "table",
			table, "value", value, "tx", (tx == nil), "reason", err.Error())
	}
	return
}

//updateByID updates a column in a table and binds the new value to the provided struct
func updateByID(tx *sqlx.Tx, column, table string, value, selection, model interface{}) (err error) {

	update := `UPDATE ` + table + ` SET ` + column +
		` = $2 WHERE id = $1 RETURNING id, ` + column

	if tx == nil {
		err = app.Db.Get(model, update, selection, value)
	} else {
		err = tx.Get(model, update, selection, value)
	}

	if err != nil {
		log.Error("failed to update value", "selection", selection,
			"value", value, "update", update, "error", err.Error())
		if tx != nil {
			tx.Rollback()
		}
	}
	return
}

//getColumnValue returns the column value of the specified table
func getColumnValue(tx *sqlx.Tx, column, table string, selection, model interface{}) (err error) {

	txWasNil := (tx == nil)
	if txWasNil {
		tx, err = app.Db.Beginx()
		if err != nil {
			log.Error("failed to begin tx", "error", err.Error())
			return
		}
	}

	stmt := `SELECT ` + column + `
	FROM ` + table + `
	WHERE id = $1`

	err = tx.Get(model, stmt, selection)
	if err != nil {
		log.Error("failed to get column value", "stmt", stmt,
			"selection", selection, "error", err.Error())
		tx.Rollback()
		return
	}

	if txWasNil {
		tx.Commit()
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

	log.Debug("generated code", "code", string(b))
	return string(b)
}
