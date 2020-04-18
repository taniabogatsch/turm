/*Package db is an intermediate between the controllers and the models. It executes
queries and thus either provides models to the controllers (select) or inserts and updates
data in the database. */
package db

import "github.com/revel/revel"

var (
	//dbLog logs all database errors
	dbLog = revel.AppLog.New("section", "database")
)
