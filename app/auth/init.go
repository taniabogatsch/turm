/*Package auth comprises all logic concerning the authentication
against the LDAP server. */
package auth

import "github.com/revel/revel"

var (
	//authLog logs all authentication errors
	authLog = revel.AppLog.New("section", "authentication")
)
