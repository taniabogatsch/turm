package models

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"
	"turm/app"

	"github.com/jmoiron/sqlx"
	"github.com/revel/revel"
)

/*Group is a model of the groups table. */
type Group struct {
	ID          int           `db:"id, primarykey, autoincrement"`
	ParentID    sql.NullInt32 `db:"parentid"`
	CourseID    sql.NullInt32 `db:"courseid"`
	Name        string        `db:"name"`
	CourseLimit sql.NullInt32 `db:"courselimit"`
	LastEditor  sql.NullInt32 `db:"lasteditor"`
	LastEdited  string        `db:"lastedited"`
	Groups      []Group       `` //not a field in the respective table

	//used to ensure unique IDs if more than one group tree is present at a page
	IDPrefix string ``
	//identifies whether any parent/child has a course limit
	InheritsLimits bool ``
	ChildHasLimits bool ``
}

/*Validate Group fields. */
func (group *Group) Validate(v *revel.Validation) {

	v.Check(group.Name,
		revel.MinSize{3},
		revel.MaxSize{255},
	).MessageKey("validation.invalid.groupName")

	if group.CourseLimit.Int32 != 0 {

		v.Check(group.ParentID.Int32,
			ParentHasCourseLimit{},
		).MessageKey("validation.invalid.courseLimit")

		v.Check(group.ID,
			ChildHasCourseLimit{},
		).MessageKey("validation.invalid.courseLimit")

		courseLimit := int(group.CourseLimit.Int32)
		v.Check(courseLimit,
			revel.Min{1},
			revel.Max{100},
		).MessageKey("validation.invalid.courseLimit")

		group.CourseLimit.Valid = true
	}

	if group.ParentID.Int32 != 0 {
		group.ParentID.Valid = true
	}

	//NOTE: the DB statement ensures that no courses are edited
}

/*Add a new group to the groups table. */
func (group *Group) Add(userIDSession *string) (err error) {

	userID, err := strconv.Atoi(*userIDSession)
	if err != nil {
		modelsLog.Error("failed to parse userID from userIDSession",
			"userIDSession", *userIDSession, "error", err.Error())
		return
	}

	err = app.Db.Get(group, stmtInsertGroup, group.ParentID, group.Name,
		group.CourseLimit, userID, time.Now().Format(revel.TimeFormats[0]))
	if err != nil {
		modelsLog.Error("failed to add group", "group", group, "userID",
			userID, "error", err.Error())
	}
	return
}

/*Edit a group of the groups table. */
func (group *Group) Edit(userIDSession *string) (err error) {

	userID, err := strconv.Atoi(*userIDSession)
	if err != nil {
		modelsLog.Error("failed to parse userID from userIDSession",
			"userIDSession", *userIDSession, "error", err.Error())
		return
	}

	err = app.Db.Get(group, stmtUpdateGroup, group.Name, group.CourseLimit, userID,
		time.Now().Format(revel.TimeFormats[0]), group.ID)
	if err != nil {
		modelsLog.Error("failed to update group", "group", group, "userID",
			userID, "error", err.Error())
	}
	return
}

/*Delete a group of the groups table. */
func (group *Group) Delete() (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		modelsLog.Error("failed to begin tx", "error", err.Error())
		return
	}

	_, err = tx.Exec(stmtMoveInactiveCourses, group.ID)
	if err != nil {
		modelsLog.Error("failed to move inactive courses", "group", group, "error", err.Error())
		tx.Rollback()
		return
	}

	_, err = tx.Exec(stmtDeleteGroup, group.ID)
	if err != nil {
		modelsLog.Error("failed to delete group", "group", group, "error", err.Error())
		tx.Rollback()
		return
	}

	tx.Commit()
	return
}

/*Groups holds all groups of the groups table. */
type Groups []Group

/*Get all groups. */
func (groups *Groups) Get(prefix *string) (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		modelsLog.Error("failed to begin tx", "error", err.Error())
		return
	}

	//get root groups for recursive calls
	err = tx.Select(groups, stmtGetRootGroups)
	if err != nil {
		modelsLog.Error("failed to get root groups", "error", err.Error())
		tx.Rollback()
		return
	}

	//start recursion
	for key := range *groups {

		(*groups)[key].IDPrefix = *prefix
		(*groups)[key].InheritsLimits = (*groups)[key].CourseLimit.Valid

		(*groups)[key].ChildHasLimits, err = getChildren(tx, &(*groups)[key])
		if err != nil {
			tx.Rollback()
			return
		}

		if (*groups)[key].CourseLimit.Valid {
			(*groups)[key].ChildHasLimits = true
		}
	}

	tx.Commit()
	return
}

/*GetByUser gets all groups created by a user. */
func (groups *Groups) GetByUser(userID *int, tx *sqlx.Tx) (err error) {

	err = tx.Select(groups, stmtSelectGroups, app.TimeZone, *userID)
	if err != nil {
		modelsLog.Error("failed to get groups", "error", err.Error())
		tx.Rollback()
	}
	return
}

//getChildren recursively returns all children of the current group.
func getChildren(tx *sqlx.Tx, group *Group) (hasLimits bool, err error) {

	err = tx.Select(&group.Groups, stmtGetChildren, group.ID)
	if err != nil {
		if err != sql.ErrNoRows {
			modelsLog.Error("failed to get children", "group", group, "error", err.Error())
			return
		}
		err = nil
	}

	//recursion
	for key := range group.Groups {

		//inherit the prefix
		group.Groups[key].IDPrefix = group.IDPrefix

		//inherit any course limit
		group.Groups[key].InheritsLimits = group.InheritsLimits
		if group.Groups[key].CourseLimit.Valid {
			group.Groups[key].InheritsLimits = true
		}

		//only get the children if the entry is a group
		if !(group.Groups[key]).CourseID.Valid {

			hasLimitsTemp, err := getChildren(tx, &group.Groups[key])
			if err != nil {
				return false, err
			}

			if group.Groups[key].CourseLimit.Valid || hasLimitsTemp {
				hasLimits = true
				group.Groups[key].ChildHasLimits = true
			}
		}
	}

	return
}

/* --- VALIDATORS --- */

/*ParentHasCourseLimit implements whether any parent group has a course limit or not. */
type ParentHasCourseLimit struct{}

/*IsSatisfied implements the validation result of ParentHasCourseLimit. */
func (courseLimit ParentHasCourseLimit) IsSatisfied(i interface{}) bool {

	parentID, parsed := i.(int32)
	if !parsed {
		return false
	}

	if parentID == 0 { //root element
		return true
	}

	var group Group
	err := app.Db.Get(&group, stmtParentHasCourseLimit, parentID)
	if err != nil {
		if err != sql.ErrNoRows {
			modelsLog.Error("failed to retrieve information for this group",
				"parentID", parentID, "error", err.Error())
		}
		return false
	}
	return !group.ParentID.Valid
}

/*DefaultMessage returns the default message of ParentHasCourseLimit. */
func (courseLimit ParentHasCourseLimit) DefaultMessage() string {
	return fmt.Sprintln("Please do not provide a course limit if any parent group already has one.")
}

/*ChildHasCourseLimit implements whether any child group has a course limit or not. */
type ChildHasCourseLimit struct{}

/*IsSatisfied implements the validation result of ChildHasCourseLimit. */
func (courseLimit ChildHasCourseLimit) IsSatisfied(i interface{}) bool {

	parentID, parsed := i.(int)
	if !parsed {
		return false
	}

	var childHasCourseLimit bool
	err := app.Db.Get(&childHasCourseLimit, stmtChildHasCourseLimit, parentID)
	if err != nil {
		modelsLog.Error("failed to retrieve information for this group",
			"parentID", parentID, "error", err.Error())
		return false
	}
	return !childHasCourseLimit
}

/*DefaultMessage returns the default message of ChildHasCourseLimit. */
func (courseLimit ChildHasCourseLimit) DefaultMessage() string {
	return fmt.Sprintln("Please do not provide a course limit if any child group already has one.")
}

/*NoActiveChildren implements whether a group contains any subgroups or active courses. */
type NoActiveChildren struct{}

/*IsSatisfied implements the validation result of NoActiveChildren. */
func (noneActive NoActiveChildren) IsSatisfied(i interface{}) bool {

	parentID, parsed := i.(int)
	if !parsed {
		return false
	}

	var noActiveChildren bool
	err := app.Db.Get(&noActiveChildren, stmtNoActiveChildren, parentID)
	if err != nil {
		modelsLog.Error("failed to retrieve information for this group",
			"parentID", parentID, "error", err.Error())
		return false
	}
	return noActiveChildren
}

/*DefaultMessage returns the default message of NoActiveChildren. */
func (noneActive NoActiveChildren) DefaultMessage() string {
	return fmt.Sprintln("Groups can only be deleted if they contain no subgroups or active courses.")
}

const (
	stmtParentHasCourseLimit = `
		WITH RECURSIVE path (id, parentid)
			AS (
				/* starting entry */
				SELECT id, parentid
				FROM groups
				WHERE id = $1
					AND courselimit IS NULL

				UNION ALL

				/* construct path */
				SELECT g.id, g.parentid
				FROM groups g, path p
				WHERE p.parentid = g.id
					AND g.courselimit IS NULL
			)

		/* select the root element of the constructed path */
		SELECT id, parentid FROM path ORDER BY parentid DESC LIMIT 1
	`

	stmtChildHasCourseLimit = `
		WITH RECURSIVE path (id, parentid, courselimit)
			AS (
				/* starting entries */
				SELECT id, parentid, courselimit
				FROM groups
				WHERE parentid = $1

				UNION ALL

				/* collect all children */
				SELECT g.id, g.parentid, g.courselimit
				FROM groups g, path p
				WHERE p.id = g.parentid
			)

		/* determine whether any child has a course limit */
		SELECT EXISTS (
			SELECT true
			FROM path
			WHERE courselimit IS NOT NULL
		) AS childHasCourseLimit
	`

	stmtNoActiveChildren = `
		SELECT NOT EXISTS (

			/* select all active courses */
			SELECT true
			FROM groups, course
			WHERE groups.courseid = course.id
				AND NOT course.active
				AND groups.parentid = $1

			UNION

			/* select all subgroups */
			SELECT true
			FROM groups
			WHERE courseid IS NULL
				AND parentid = $1

		) AS noActiveChildren
	`

	//TODO: order: first groups by name, then courses by name, both ascending
	//TODO: only show active courses
	//TODO: union courses to get only active courses
	stmtGetChildren = `
		SELECT id, parentid, courseid, name, courselimit
		FROM groups
		WHERE parentid = $1
	`

	stmtInsertGroup = `
		INSERT INTO groups
			(parentid, name, courselimit, lasteditor, lastedited)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name
	`

	stmtUpdateGroup = `
		UPDATE groups
		SET name = $1, courselimit = $2, lasteditor = $3, lastedited = $4
		WHERE id = $5
			AND courseid IS NULL /* courses must be edited differently */
		RETURNING id, name
	`

	stmtMoveInactiveCourses = `
		UPDATE groups
		SET parentid = (
			SELECT parentid FROM groups WHERE id = $1
		) WHERE parentid = $1
	`

	stmtDeleteGroup = `
		DELETE FROM groups
		WHERE id = $1
			AND courseid IS NULL
	`

	stmtGetRootGroups = `
		SELECT id, parentid, courseid, name, courselimit
		FROM groups
		WHERE parentid IS NULL
		ORDER BY name ASC
	`

	stmtSelectGroups = `
		SELECT
			id, parentid, name, courselimit,
			TO_CHAR (lastedited AT TIME ZONE $1, 'YYYY-MM-DD HH24:MI') as lastedited
		FROM groups
		WHERE lasteditor = $2
			AND courseid IS NULL
		ORDER BY name ASC
	`
)
