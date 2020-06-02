/*Package auth comprises all logic concerning the authentication
against the LDAP server. */
package auth

import "github.com/revel/revel"

var (
	//log all authentication errors
	log = revel.AppLog.New("section", "authentication")
)
