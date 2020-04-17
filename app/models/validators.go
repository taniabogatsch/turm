package models

import (
	"fmt"
	"turm/app"

	"github.com/revel/revel"
)

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
}

/*IsSatisfied implements the validation result of Unique. */
func (uniqueV Unique) IsSatisfied(i interface{}) bool {

	var unique bool
	data, parsed := i.(ValidateUniqueData)
	if !parsed {
		return false
	}

	selectExists := `SELECT NOT EXISTS (SELECT ` + data.Column +
		` FROM ` + data.Table + ` WHERE ` + data.Column + ` = $1) AS unique`
	err := app.Db.Get(&unique, selectExists, data.Value)
	if err != nil {
		revel.AppLog.Error("failed to retrieve information about this column",
			"SQL", selectExists, "data", data, "error", err.Error())
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
