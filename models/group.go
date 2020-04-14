package models

import (
	"database/sql"

	"github.com/revel/revel"
)

/*Group contains all directly group related values. */
type Group struct {
	ID           int           `db:"id, primarykey, autoincrement"`
	ParentID     sql.NullInt32 `db:"parentid"`
	CourseID     sql.NullInt32 `db:"courseid"`
	Name         string        `db:"name"`
	MaxCourses   sql.NullInt32 `db:"maxcourses"`
	Creator      sql.NullInt32 `db:"creator"`
	CreationDate string        `db:"creationdate"`
}

/*ValidateGroup validates the Group struct fields. */
func (group *Group) ValidateGroup(v *revel.Validation) {
	//TODO
}
