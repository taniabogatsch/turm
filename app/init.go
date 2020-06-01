package app

import (
	"errors"
	"turm/modules/jobs/app/jobs"

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
)

//init application
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

	revel.TimeFormats = append(revel.TimeFormats, "2006-01-02 15:04:05 -0700")

	//register startup functions with OnAppStart
	revel.OnAppStart(initPasswords) //NOTE: must be executed first
	revel.OnAppStart(initConfigVariables)
	revel.OnAppStart(initDB)

	//register scheduled jobs
	revel.OnAppStart(initJobSchedules)
	revel.OnAppStart(func() {
		jobs.Schedule("@every 30s", sendEMails{})
		jobs.Schedule(jobSchedules["jobs.dbbackup"], backupDB{})
	})

	//register custom template functions
	revel.TemplateFuncs["dict_addLocale"] = func(locale string, values ...interface{}) (map[string]interface{}, error) {
		if len(values)%2 != 0 {
			return nil, errors.New("invalid dict call")
		}
		dict := make(map[string]interface{}, len(values)/2)
		for i := 0; i < len(values); i += 2 {
			key, ok := values[i].(string)
			if !ok {
				return nil, errors.New("dict keys must be strings")
			}
			dict[key] = values[i+1]
		}
		dict["currentLocale"] = locale
		return dict, nil
	}
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
