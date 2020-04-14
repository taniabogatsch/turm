package app

import (
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/revel/revel"

	//Blank import needed for loading Postgres driver for SQLx
	_ "github.com/jackc/pgx/stdlib"
)

/*DB is an abstraction to the actual database connection. */
type DB struct {
	*sqlx.DB
}

var (
	//AppVersion revel app version (ldflags)
	AppVersion string

	//BuildTime revel app build-time (ldflags)
	BuildTime string

	//Db is the database object representing the DB connections
	Db *DB

	//TimeZone of the application
	TimeZone string
)

func init() {
	//Filters is the default set of global filters.
	revel.Filters = []revel.Filter{
		revel.PanicFilter,             //Recover from panics and display an error page instead.
		revel.RouterFilter,            //Use the routing table to select the right Action
		revel.FilterConfiguringFilter, //A hook for adding or removing per-Action filters.
		revel.ParamsFilter,            //Parse parameters into Controller.Params.
		revel.SessionFilter,           //Restore and write the session cookie.
		revel.FlashFilter,             //Restore and write the flash cookie.
		revel.ValidationFilter,        //Restore kept validation errors and save new ones from cookie.
		revel.I18nFilter,              //Resolve the requested language
		HeaderFilter,                  //Add some security based headers
		revel.InterceptorFilter,       //Run interceptors around the action.
		revel.CompressFilter,          //Compress the result.
		revel.BeforeAfterFilter,       //Call the before and after filter functions
		revel.ActionInvoker,           //Invoke the action.
	}

	//Register startup functions with OnAppStart
	revel.OnAppStart(initDB)
	revel.OnAppStart(initConfigVariables)
}

//HeaderFilter adds common security headers
//There is a full implementation of a CSRF filter in
//https://github.com/revel/modules/tree/master/csrf
var HeaderFilter = func(c *revel.Controller, fc []revel.Filter) {
	c.Response.Out.Header().Add("X-Frame-Options", "SAMEORIGIN")
	c.Response.Out.Header().Add("X-XSS-Protection", "1; mode=block")
	c.Response.Out.Header().Add("X-Content-Type-Options", "nosniff")
	c.Response.Out.Header().Add("Referrer-Policy", "strict-origin-when-cross-origin")

	fc[0](c, fc[1:]) //Execute the next filter stage.
}

/*initDB sets up a database connection. */
func initDB() {

	revel.AppLog.Info("init DB")

	driver := revel.Config.StringDefault("db.driver", "postgres")
	conn, found := revel.Config.String("db.connection")
	if !found {
		revel.AppLog.Error("cannot find key in config", "key", "db.connection")
		os.Exit(10)
	}

	db, err := sqlx.Connect(driver, conn)
	if err != nil {
		revel.AppLog.Error("DB connection error", "error", err.Error())
		os.Exit(10)
	}

	Db = &DB{DB: db}

	//validate the connection
	var dummy int
	err = Db.Get(&dummy, `select 1 as dummy`)
	if err != nil {
		revel.AppLog.Error("DB connection not working", "error", err.Error())
		os.Exit(1)
	}

	revel.AppLog.Info("connected to DB")
}

/*initConfigVariables initializes all config variables used in the application. */
func initConfigVariables() {

	revel.AppLog.Info("init custom config variables")

	var found bool
	if TimeZone, found = revel.Config.String("timezone.long"); !found {
		revel.AppLog.Error("cannot find key in config", "key", "timezone.long")
		os.Exit(1)
	}
}
