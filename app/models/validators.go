package models

import (
	"fmt"
	"reflect"
	"turm/app"
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
