package app

import (
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/revel/revel"
)

var (
	//DBData holds all DB connection data
	DBData DBConn
)

/*DBConn contains all DB connection fields. */
type DBConn struct {
	User     string
	Name     string
	Host     string
	Port     string
	Password string
}

//initDB sets up a database connection.
func initDB() {

	revel.AppLog.Info("init DB")

	driver := revel.Config.StringDefault("db.driver", "postgres")
	conn := "user=" + DBData.User + " password=" + DBData.Password + " dbname=" +
		DBData.Name + " host=" + DBData.Host + " port=" + DBData.Port + " sslmode=disable"

	db, err := sqlx.Connect(driver, conn)
	if err != nil {
		revel.AppLog.Fatal("DB connection error", "driver", driver, "conn",
			conn, "error", err.Error())
	}

	Db = &DB{DB: db}

	//validate the connection
	var dummy int
	err = Db.Get(&dummy, `select 1 as dummy`)
	if err != nil {
		revel.AppLog.Fatal("DB connection not working", "error", err.Error())
	}

	revel.AppLog.Info("connected to DB")
}

//initDBData initializes all DB config variables
func initDBData() {

	var found bool
	if DBData.Host, found = revel.Config.String("db.host"); !found {
		revel.AppLog.Error("no db.host set in config")
		os.Exit(1)
	}
	if DBData.Port, found = revel.Config.String("db.port"); !found {
		revel.AppLog.Error("no db.port set in config")
		os.Exit(1)
	}
	if DBData.User, found = revel.Config.String("db.user"); !found {
		revel.AppLog.Error("no db.user set in config")
		os.Exit(1)
	}
	if DBData.Name, found = revel.Config.String("db.db"); !found {
		revel.AppLog.Error("no db.db set in config")
		os.Exit(1)
	}
}
