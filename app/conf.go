package app

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/revel/revel"
)

/*ServerConn contains all server connection fields. */
type ServerConn struct {
	Address string
	Port    string
	SSL     string
	URL     string
}

//general config variables
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

	//Server holds all server connection data
	Server ServerConn

	//passwords holds all passwords used in the application
	passwords map[string]string
)

//initConfigVariables initializes all config variables used in the application.
func initConfigVariables() {

	revel.AppLog.Info("init custom config variables")
	var found bool
	var err error

	initMailerData()
	initDBData()
	initServerData()
	initCloudData() //NOTE: must be after initMailerData

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

	passwords = make(map[string]string)
	json.Unmarshal(fileContent, &passwords)

	Mailer.Password = passwords["email.pw"]
	DBData.Password = passwords["db.pw"]
}

//initServerData initializes all server config variables
func initServerData() {

	var found bool
	if Server.Address, found = revel.Config.String("http.addr"); !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "http.addr")
	}
	if Server.Port, found = revel.Config.String("http.port"); !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "http.port")
	}
	if Server.SSL, found = revel.Config.String("http.ssl"); !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "http.ssl")
	}
	Server.URL = Server.Address
	if Server.Address != "localhost" {
		if Server.SSL == "true" {
			Server.URL = "https://" + Server.Address
		} else {
			Server.URL = "http://" + Server.Address
		}
	}
	if (Server.Port != "80" && Server.Port != "443") || Server.Port == "" {
		Server.URL = Server.URL + ":" + Server.Port
	}
}
