/*Package models contains all database tables as structs and their validation
functions. It also contains additional structs representing front end data, such
as the user login credentials.*/
package models

import (
	"turm/app"

	"github.com/jmoiron/sqlx"
	"github.com/revel/revel"
)

var (
	//log all model errors
	log = revel.AppLog.New("section", "models")
)

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
			"value", value, "error", err.Error())
		if tx != nil {
			tx.Rollback()
		}
	}
	return
}
