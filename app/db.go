package app

import (
	"github.com/jmoiron/sqlx"
	"github.com/revel/revel"
)

/*DB is an abstraction to the actual database connection. */
type DB struct {
	*sqlx.DB
}

var (
	//dbData holds all DB connection data
	dbData dbConn

	//Db is the database object representing the DB connections
	Db *DB
)

//dbConn contains all DB connection fields
type dbConn struct {
	User     string
	Name     string
	Host     string
	Port     string
	Password string
	Driver   string
}

//initDB sets up a database connection.
func initDB() {

	revel.AppLog.Info("init DB")

	conn := "user=" + dbData.User + " password=" + dbData.Password + " dbname=" +
		dbData.Name + " host=" + dbData.Host + " port=" + dbData.Port + " sslmode=disable"

	db, err := sqlx.Connect(dbData.Driver, conn)
	if err != nil {
		revel.AppLog.Fatal("DB connection error", "driver", dbData.Driver, "conn",
			conn, "error", err.Error())
	}

	Db = &DB{DB: db}

	// force a connection and test that it worked
	err = Db.Ping()
	if err != nil {
		revel.AppLog.Fatal("DB connection test failed", "err", err.Error())
	}

	revel.AppLog.Info("connected to DB")
}

//initDBData initializes all DB config variables
func initDBData() {

	var found bool
	if dbData.Host, found = revel.Config.String("db.host"); !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "db.host")
	}
	if dbData.Port, found = revel.Config.String("db.port"); !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "db.port")
	}
	if dbData.User, found = revel.Config.String("db.user"); !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "db.user")
	}
	if dbData.Name, found = revel.Config.String("db.db"); !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "db.db")
	}
	if dbData.Driver, found = revel.Config.String("db.driver"); !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "db.driver")
	}
}
