package models

import (
	"fmt"
	"reflect"
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

/*NotRequiredValidator implements the validation of fields that must not be set. */
type NotRequiredValidator struct{}

/*IsSatisfied implements the validation result of NotRequiredValidator. */
func (v NotRequiredValidator) IsSatisfied(i interface{}) bool {

	if i == nil {
		return true
	}
	switch value := reflect.ValueOf(i); value.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String, reflect.Chan:
		if value.Len() == 0 {
			return true
		}
	case reflect.Ptr:
		return v.IsSatisfied(reflect.Indirect(value).Interface())
	}
	return reflect.DeepEqual(i, reflect.Zero(reflect.TypeOf(i)).Interface())
}

/*DefaultMessage returns the default message of NotRequiredValidator. */
func (v NotRequiredValidator) DefaultMessage() string {
	return fmt.Sprintln("Please do not provide this value.")
}

/*UniqueValidator implements the validation of the uniqueness of a column value in a provided table. */
type UniqueValidator struct{}

/*ValidateUniqueData contains all data to validate the uniqueness of a column value in a table. */
type ValidateUniqueData struct {
	Column string
	Table  string
	Value  string
}

/*IsSatisfied implements the validation result of UniqueValidator. */
func (uniqueV UniqueValidator) IsSatisfied(i interface{}) bool {

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

/*DefaultMessage returns the default message of UniqueValidator. */
func (uniqueV UniqueValidator) DefaultMessage() string {
	return fmt.Sprintln("Please provide a unique value.")
}
