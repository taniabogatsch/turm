package models

import (
	"database/sql"

	"github.com/revel/revel"
)

/*Event contains all directly event related values. */
type Event struct {
	ID            int            `db:"id, primarykey, autoincrement"`
	CourseID      int            `db:"courseid"`
	Capacity      int            `db:"capacity"`
	HasWaitlist   bool           `db:"haswaitlist"`
	Title         string         `db:"title"`
	Description   sql.NullString `db:"description"`
	EnrollmentKey sql.NullString `db:"enrollmentkey"`
	Meetings      []Meeting      ``
}

/*Validate validates the Event struct fields. */
func (event *Event) Validate(v *revel.Validation) {
	//TODO
}
