package auth

import (
	"crypto/tls"
	"fmt"
	"strconv"
	"strings"
	"turm/app"
	"turm/app/models"

	"github.com/revel/revel"
	ldap "gopkg.in/ldap.v2"
)

/*LDAPServerAuth implements the authentication of an user against the ldap server after the user
entered his username and password. */
func LDAPServerAuth(credentials *models.Credentials, user *models.User) (err error) {

	//get a TLS encrypted connection
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	hostAndPort := fmt.Sprintf("%s:%d", app.LdapHost, app.LdapPort)
	l, err := ldap.DialTLS("tcp", hostAndPort, tlsConfig)
	if err != nil {
		revel.AppLog.Error("error getting the TLS encrypted connection",
			"hostAndPort", hostAndPort, "tlsConfig", tlsConfig, "error", err.Error())
		return
	}
	defer l.Close()

	//try to bind with specified user
	base := fmt.Sprintf("cn=%s,ou=user,o=uni", credentials.Username)
	err = l.Bind(base, credentials.Password) //actual 'login'
	if err != nil {
		if !strings.Contains(err.Error(), "Invalid Credentials") {
			revel.AppLog.Error("cannot login the user", "base", base, "error", err.Error())
		}
		return
	}

	//at this point the actual login was successful
	//now we want to get the user details

	//attrNames is used to filter for specific attributes
	attrNames := []string{"thuEduStudentNumber", "givenName", "sn", "mail", "thuEduTitle",
		"thuEduSalutation", "eduPersonAffiliation", "thuEduAcademicTitle", "thuEduNameExtension"}

	//search for the given username
	searchRequest := ldap.NewSearchRequest(
		base,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		fmt.Sprintf("(&(objectClass=user)(uid=%s))", credentials.Username), //we are looking for a user
		attrNames, //attrNames to get only certain ones
		nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil {
		revel.AppLog.Error("error getting attributes", "search request", searchRequest, "error", err.Error())
		return
	}
	//must be at least one, because we already logged in with this username
	if len(sr.Entries) != 1 {
		revel.AppLog.Error("user does not exist or too many entries returned")
		return err
	}

	//get the entry
	e := sr.Entries[0]

	//now we simly put the data we searched for with attrNames into an user struct
	user.FirstName = e.GetAttributeValue("givenName")
	user.LastName = e.GetAttributeValue("sn")
	user.EMail = strings.ToLower(e.GetAttributeValue("mail"))
	user.Affiliations = e.GetAttributeValues("eduPersonAffiliation")

	switch salutation := e.GetAttributeValue("thuEduSalutation"); salutation {
	case "Frau":
		user.Salutation = models.MS
	case "Herr":
		user.Salutation = models.MR
	default:
		user.Salutation = models.NONE
	}

	if e.GetAttributeValue("thuEduTitle") != "" {
		user.Title.String = e.GetAttributeValue("thuEduTitle")
		user.Title.Valid = true
	}
	if e.GetAttributeValue("thuEduAcademicTitle") != "" {
		user.AcademicTitle.String = e.GetAttributeValue("thuEduAcademicTitle")
		user.AcademicTitle.Valid = true
	}
	if e.GetAttributeValue("thuEduNameExtension") != "" {
		user.NameAffix.String = e.GetAttributeValue("thuEduNameExtension")
		user.NameAffix.Valid = true
	}

	//set the matriculation number, if not null
	if e.GetAttributeValue("thuEduStudentNumber") != "" {
		matrNr, err := strconv.Atoi(e.GetAttributeValue("thuEduStudentNumber"))
		if err != nil {
			revel.AppLog.Error("error parsing matriculation number",
				"matrNr", e.GetAttributeValue("thuEduStudentNumber"), "error", err.Error())
			return err
		}
		user.MatrNr.Int32 = int32(matrNr)
		user.MatrNr.Valid = true
	}

	return
}
