package models

import "turm/app"

//updateByID a column in a table and bind the new value to the provided struct
func updateByID(column string, value interface{}, selection interface{}, table string, model interface{}) (err error) {

	update := `UPDATE ` + table + ` SET ` + column + ` = $2 WHERE id = $1 RETURNING id, ` + column

	err = app.Db.Get(model, update, selection, value)
	if err != nil {
		log.Error("failed to update value", "selection", selection,
			"value", value, "error", err.Error())
	}
	return
}
