/*Package models contains all database tables as structs and their validation
functions. It also contains additional structs representing front end data, such
as the user login credentials.*/
package models

import "github.com/revel/revel"

var (
	//log all model errors
	log = revel.AppLog.New("section", "models")
)
