package models

import (
	"database/sql/driver"
	"errors"
	"strings"
)

/*Affiliations contains all affiliations of a user. */
type Affiliations []string

/*NullAffiliations represents affiliations that may be null. */
type NullAffiliations struct {
	Affiliations Affiliations
	Valid        bool //Valid is true if Affiliations is not NULL
}

/*Value constructs a SQL Value from NullAffiliations. */
func (affiliations NullAffiliations) Value() (driver.Value, error) {

	if !affiliations.Valid {
		return nil, nil
	}

	var str string
	for _, affiliation := range affiliations.Affiliations {
		str += `"` + affiliation + `",`
	}
	return driver.Value("{" + strings.TrimRight(str, ",") + "}"), nil
}

/*Scan constructs NullAffiliations from a SQL Value. */
func (affiliations *NullAffiliations) Scan(value interface{}) error {

	if value == nil {
		affiliations.Affiliations = []string{""}
		affiliations.Valid = false
		return nil
	}

	affiliations.Valid = true

	switch value.(type) {
	case string:
		str := value.(string)
		str = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(str, "{", ""), "}", ""))
		affiliations.Affiliations = strings.Split(str, ",")
	default:
		return errors.New("incompatible type for Affiliations")
	}
	return nil
}
