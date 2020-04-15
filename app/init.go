package app

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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

//config variables
var (
	//TimeZone of the application
	TimeZone string

	//DefaultLanguage is the default language of the page
	DefaultLanguage string
	//Languages holds all languages supported by the application
	Languages []string

	//LdapHost is the host of the LDAP server
	LdapHost string
	//LdapPort is the port of the LDAP server
	LdapPort int

	//ServiceEMail is the service e-mail-address of the application
	ServiceEMail string
	//EMailServer is the service e-mail server
	EMailServer string
	//EMailURL is the URL of the e-mail server
	EMailURL string
	//EMailUser is the service e-mail server user
	EMailUser string
	//EMailSuffix determines whether an user cannot register but has to use the university login
	EMailSuffix string

	//HTTPAddr is the server http address
	HTTPAddr string
	//Port is the server port
	Port string
	//SSL defines whether SSL is used or not
	SSL string
	//URL is the server URL to be used in e-mails
	URL string

	//Passwords holds all passwords used in the application
	Passwords map[string]string
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

	//Register startup functions with OnAppStart
	revel.OnAppStart(initConfigVariables)
	revel.OnAppStart(initPasswords)
	revel.OnAppStart(initDB)
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

//initPasswords initializes all passwords.
func initPasswords() {

	revel.AppLog.Info("init passwords")

	//open json file
	filepath := filepath.Join(revel.BasePath, "conf", "passwords.json")
	jsonFile, err := os.Open(filepath)
	if err != nil {
		revel.AppLog.Fatal("cannot open file", "filepath", filepath, "error", err.Error())
	}
	defer jsonFile.Close()

	//read the file content
	fileContent, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		revel.AppLog.Fatal("cannot read file", "jsonFile", jsonFile, "error", err.Error())
	}

	Passwords = make(map[string]string)
	json.Unmarshal(fileContent, &Passwords)
}

//initConfigVariables initializes all config variables used in the application.
func initConfigVariables() {

	revel.AppLog.Info("init custom config variables")
	var found bool
	var err error

	//time zone
	if TimeZone, found = revel.Config.String("timezone.long"); !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "timezone.long")
	}

	//languages
	if DefaultLanguage, found = revel.Config.String("i18n.default_language"); !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "i18n.default_language")
	}
	var languageList string
	if languageList, found = revel.Config.String("languages.list"); !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "languages.list")
	}
	Languages = strings.Split(languageList, ", ")

	//ldap config
	if LdapHost, found = revel.Config.String("ldap.host"); !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "ldap.host")
	}
	var portStr string
	if portStr, found = revel.Config.String("ldap.port"); !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "ldap.port")
	}
	if LdapPort, err = strconv.Atoi(portStr); err != nil {
		revel.AppLog.Fatal("invalid ldap.port value set in config", "value", portStr, "error", err.Error())
	}

	//e-mail
	if ServiceEMail, found = revel.Config.String("email.email"); !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "email.email")
	}
	if EMailServer, found = revel.Config.String("email.server"); !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "email.server")
	}
	var port string
	if port, found = revel.Config.String("email.port"); !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "email.port")
	}
	EMailURL = EMailServer + ":" + port
	if EMailUser, found = revel.Config.String("email.user"); !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "email.user")
	}
	if EMailSuffix, found = revel.Config.String("email.suffix"); !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "email.suffix")
	}

	//server setup variables
	if HTTPAddr, found = revel.Config.String("http.addr"); !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "http.addr")
	}
	if Port, found = revel.Config.String("http.port"); !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "http.port")
	}
	if SSL, found = revel.Config.String("http.ssl"); !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "http.ssl")
	}
	URL = HTTPAddr
	if HTTPAddr != "localhost" {
		if SSL == "true" {
			URL = "https://" + HTTPAddr
		} else {
			URL = "http://" + HTTPAddr
		}
	}
	if (Port != "80" && Port != "443") || Port == "" {
		URL = URL + ":" + Port
	}
}

//initDB sets up a database connection.
func initDB() {

	revel.AppLog.Info("init DB")

	driver := revel.Config.StringDefault("db.driver", "postgres")
	conn, found := revel.Config.String("db.connection")
	if !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "db.connection")
	}

	db, err := sqlx.Connect(driver, conn)
	if err != nil {
		revel.AppLog.Fatal("DB connection error", "driver", driver, "conn", conn, "error", err.Error())
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
