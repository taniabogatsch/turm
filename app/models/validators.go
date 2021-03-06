package models

//more specific validators are in the respective model files

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
	"turm/app"

	"github.com/jmoiron/sqlx"
	"github.com/revel/revel"
)

/*ValidateLength of a not null string. */
func ValidateLength(str *string, msgKey string, min, max int, v *revel.Validation) {

	*str = strings.TrimSpace(*str)
	v.MinSize(*str, min).MessageKey(msgKey)
	v.MaxSize(*str, max).MessageKey(msgKey)
}

/*ValidateLengthAndValid sets valid to true if the string is not empty and
also validates its length (if non-empty). */
func ValidateLengthAndValid(str *sql.NullString, msgKey string, min, max int, v *revel.Validation) {

	(*str).String = strings.TrimSpace((*str).String)
	if len((*str).String) > 0 {
		(*str).Valid = true
		s := (*str).String //workaround for revel error
		v.MaxSize(s, max).MessageKey(msgKey)
	}
}

/*LanguageValidator implements the validation of the selected language. */
type LanguageValidator struct{}

/*IsSatisfied implements the validation result of LanguageValidator. */
func (v LanguageValidator) IsSatisfied(i interface{}) bool {

	data, parsed := i.(string)
	if !parsed {
		return false
	}

	for _, language := range app.Languages {
		if data == language {
			return true
		}
	}
	return false
}

/*DefaultMessage returns the default message of LanguageValidator. */
func (v LanguageValidator) DefaultMessage() string {
	return fmt.Sprintln("Please provide a valid language.")
}

/*NotRequired implements the validation of fields that must not be set. */
type NotRequired struct{}

/*IsSatisfied implements the validation result of NotRequired. */
func (v NotRequired) IsSatisfied(i interface{}) bool {
	return !revel.Required{}.IsSatisfied(i)
}

/*DefaultMessage returns the default message of NotRequired. */
func (v NotRequired) DefaultMessage() string {
	return fmt.Sprintln("Please do not provide this value.")
}

/*Unique implements the validation of the uniqueness of a column value in a provided table. */
type Unique struct{}

/*ValidateUniqueData contains all data to validate the uniqueness of a column value in a table. */
type ValidateUniqueData struct {
	Column string
	Table  string
	Value  string
	Tx     *sqlx.Tx
}

/*IsSatisfied implements the validation result of Unique. */
func (uniqueV Unique) IsSatisfied(i interface{}) bool {

	var unique bool
	data, parsed := i.(ValidateUniqueData)
	if !parsed {
		return false
	}

	stmt := `SELECT NOT EXISTS (SELECT ` + data.Column +
		` FROM ` + data.Table + ` WHERE ` + data.Column + ` = $1) AS unique`

	err := data.Tx.Get(&unique, stmt, data.Value)
	if err != nil {
		log.Error("failed to retrieve information about this column",
			"stmt", stmt, "data", data, "error", err.Error())
		data.Tx.Rollback()
		return false
	}

	return unique
}

/*DefaultMessage returns the default message of Unique. */
func (uniqueV Unique) DefaultMessage() string {
	return fmt.Sprintln("Please provide a unique value.")
}

/*NotUnique implements the validation of fields that must not be set. */
type NotUnique struct{}

/*IsSatisfied implements the validation result of NotUnique. */
func (v NotUnique) IsSatisfied(i interface{}) bool {
	return !Unique{}.IsSatisfied(i)
}

/*DefaultMessage returns the default message of NotUnique. */
func (v NotUnique) DefaultMessage() string {
	return fmt.Sprintln("Please provide a non-unique value.")
}

/*IsTimestamp validates if a value can be parsed to a timestamp. */
type IsTimestamp struct{}

/*IsSatisfied implements the validation result of IsTimestamp. */
func (v IsTimestamp) IsSatisfied(i interface{}) bool {

	timestamp, parsed := i.(string)
	if !parsed {
		return false
	}

	loc, _ := time.LoadLocation(app.TimeZone)
	_, err := time.ParseInLocation("2006-01-02 15:04", timestamp, loc)
	if err != nil {
		return false
	}
	return true
}

/*DefaultMessage returns the default message of IsTimestamp. */
func (v IsTimestamp) DefaultMessage() string {
	return fmt.Sprintln("Please provide a valid date and time format.")
}
