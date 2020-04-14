package models

import (
	"fmt"
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
