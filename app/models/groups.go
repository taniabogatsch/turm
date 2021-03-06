package models

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
	"turm/app"

	"github.com/jmoiron/sqlx"
	"github.com/revel/revel"
)

/*Group is a model of the groups table. Groups are used to specify sections
and subsections of an institution in which courses are placed. */
type Group struct {
	ID          int           `db:"id, primarykey, autoincrement"`
	ParentID    sql.NullInt32 `db:"parent_id"`
	Name        string        `db:"name"`
	CourseLimit sql.NullInt32 `db:"course_limit"`
	LastEditor  sql.NullInt32 `db:"last_editor"`
	LastEdited  string        `db:"last_edited"`
	Groups      []Group       `` //not a field in the respective table

	//used to ensure unique IDs if more than one group tree is present at a page
	IDPrefix string ``
	//identifies whether any parent/child has a course limit
	InheritsLimits bool ``
	ChildHasLimits bool ``
	//used to open courses
	CourseID int `db:"course_id"`
}

/*Validate Group fields. */
func (group *Group) Validate(v *revel.Validation) {

	group.Name = strings.TrimSpace(group.Name)
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
}

/*Insert a new group into the groups table. */
func (group *Group) Insert(userID *int) (err error) {

	err = app.Db.Get(group, stmtInsertGroup, group.ParentID, group.Name,
		group.CourseLimit, userID, time.Now().Format(revel.TimeFormats[0]))
	if err != nil {
		log.Error("failed to insert group", "group", group, "userID",
			userID, "error", err.Error())
	}
	return
}

/*Update a group in the groups table. */
func (group *Group) Update(userID *int) (err error) {

	err = app.Db.Get(group, stmtUpdateGroup, group.Name, group.CourseLimit, userID,
		time.Now().Format(revel.TimeFormats[0]), group.ID)
	if err != nil {
		log.Error("failed to update group", "group", group, "userID",
			userID, "error", err.Error())
	}
	return
}

/*Delete a group from the groups table. */
func (group *Group) Delete() (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	_, err = tx.Exec(stmtMoveInactiveCourses, group.ID)
	if err != nil {
		log.Error("failed to move inactive courses", "group", group, "error", err.Error())
		tx.Rollback()
		return
	}

	if err = deleteByID("id", "groups", group.ID, tx); err != nil {
		return
	}

	tx.Commit()
	return
}

/*Get a group by its ID. */
func (group *Group) Get(tx *sqlx.Tx) (err error) {

	err = tx.Get(group, stmtGetGroup, group.ID)
	if err != nil {
		log.Error("failed to get group", "group", group, "error", err.Error())
		tx.Rollback()
	}
	return
}

/*Groups holds all groups in which courses can be placed. The struct is
filled recursively by traversing the children of each group. */
type Groups []Group

/*Select all groups. */
func (groups *Groups) Select(prefix *string) (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	//get root groups for recursive calls
	err = tx.Select(groups, stmtGetRootGroups)
	if err != nil {
		log.Error("failed to select root groups", "error", err.Error())
		tx.Rollback()
		return
	}

	//start recursion
	for key := range *groups {

		(*groups)[key].IDPrefix = *prefix
		(*groups)[key].InheritsLimits = (*groups)[key].CourseLimit.Valid

		(*groups)[key].ChildHasLimits, err = (&(*groups)[key]).selectChildren(tx)
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

/*SelectPath selects the path of a course in the groups tree. */
func (groups *Groups) SelectPath(courseID *int, tx *sqlx.Tx) (err error) {

	if tx == nil {
		err = app.Db.Select(groups, stmtGetPath, *courseID)
	} else {
		err = tx.Select(groups, stmtGetPath, *courseID)
	}

	if err != nil {
		log.Error("failed to select path", "courseID", *courseID,
			"error", err.Error())
		if tx != nil {
			tx.Rollback()
		}
	}
	return
}

/*SelectByUser selects all groups created by a user. */
func (groups *Groups) SelectByUser(userID *int, tx *sqlx.Tx) (err error) {

	err = tx.Select(groups, stmtSelectGroups, app.TimeZone, *userID)
	if err != nil {
		log.Error("failed to select groups by user", "error", err.Error())
		tx.Rollback()
	}
	return
}

//selectChildren recursively returns all children of the current group.
func (group *Group) selectChildren(tx *sqlx.Tx) (hasLimits bool, err error) {

	err = tx.Select(&group.Groups, stmtGetChildren, group.ID)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Error("failed to select children", "group", group, "error", err.Error())
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
		if group.Groups[key].ID != 0 {

			hasLimitsTemp, err := (&group.Groups[key]).selectChildren(tx)
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

	var limit int
	err := app.Db.Get(&limit, stmtParentsGetCourseLimit, parentID)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Error("failed to retrieve information for this group",
				"parentID", parentID, "error", err.Error())
		}
		return true
	}
	return false
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
		log.Error("failed to retrieve information for this group",
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
		log.Error("failed to retrieve information for this group",
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
	stmtParentsGetCourseLimit = `
		WITH RECURSIVE path (parent_id, id, course_limit)
			AS (
				/* starting entry */
				SELECT parent_id, id, course_limit
				FROM groups
				WHERE id = $1

				UNION ALL

				/* construct path */
				SELECT g.parent_id, g.id, g.course_limit
				FROM groups g, path p
				WHERE p.parent_id = g.id
			)

		/* select the course limit */
		SELECT course_limit AS limit
		FROM path
		WHERE course_limit IS NOT NULL
	`

	stmtChildHasCourseLimit = `
		WITH RECURSIVE path (id, parent_id, course_limit)
			AS (
				/* starting entries */
				SELECT id, parent_id, course_limit
				FROM groups
				WHERE parent_id = $1

				UNION ALL

				/* collect all children */
				SELECT g.id, g.parent_id, g.course_limit
				FROM groups g, path p
				WHERE p.id = g.parent_id
			)

		/* determine whether any child has a course limit */
		SELECT EXISTS (
			SELECT true
			FROM path
			WHERE course_limit IS NOT NULL
		) AS child_has_course_limit
	`

	stmtNoActiveChildren = `
		SELECT NOT EXISTS (

			/* select all active courses */
			SELECT true
			FROM groups, courses
			WHERE groups.id = courses.parent_id
				AND NOT courses.active
				AND groups.id = $1

			UNION

			/* select all subgroups */
			SELECT true
			FROM groups
			WHERE parent_id = $1

		) AS no_active_children
	`

	stmtGetChildren = `
		/* get all groups */
		(
			SELECT id, parent_id, name::text AS name, course_limit, id AS course_id
			FROM groups
			WHERE parent_id = $1
			ORDER BY name ASC
		)

		UNION ALL

		/* get all courses */
		(
			SELECT 0 AS id, co.parent_id, co.title::text AS name,
				(
					SELECT g.course_limit
					FROM groups g, courses c
					WHERE g.id = c.parent_id
						AND g.id = $1
						AND c.id = co.id
				) AS course_limit, co.id AS course_id

			FROM courses co
			WHERE co.parent_id = $1
				AND co.active
				AND (current_timestamp < co.expiration_date)
			ORDER BY name ASC
		)
	`

	stmtInsertGroup = `
		INSERT INTO groups
			(parent_id, name, course_limit, last_editor, last_edited)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name
	`

	stmtUpdateGroup = `
		UPDATE groups
		SET name = $1, course_limit = $2, last_editor = $3, last_edited = $4
		WHERE id = $5
		RETURNING id, name
	`

	stmtMoveInactiveCourses = `
		UPDATE courses
		SET parent_id = (
			SELECT parent_id FROM groups WHERE id = $1
		) WHERE parent_id = $1
	`

	stmtGetRootGroups = `
		SELECT id, parent_id, name, course_limit
		FROM groups
		WHERE parent_id IS NULL
		ORDER BY name ASC
	`

	stmtSelectGroups = `
		SELECT
			id, parent_id, name, course_limit,
			TO_CHAR (last_edited AT TIME ZONE $1, 'YYYY-MM-DD HH24:MI') as last_edited
		FROM groups
		WHERE last_editor = $2
		ORDER BY name ASC
	`

	stmtGetPath = `
		WITH RECURSIVE path (parent_id, name, id)
			AS (
				/* starting entry */
				SELECT parent_id, title::text AS name,
					(
						SELECT MAX(id) + 1
						FROM groups
					) AS id
				FROM courses
				WHERE id = $1
					AND parent_id IS NOT NULL

				UNION ALL

				/* construct path */
				SELECT g.parent_id, g.name::text AS name, g.id
				FROM groups g, path p
				WHERE p.parent_id = g.id
			)

		/* get all path entries */
		SELECT id, parent_id, name FROM path ORDER BY id ASC
	`

	stmtGetGroup = `
		SELECT
			id, parent_id, name, course_limit
		FROM groups
		WHERE id = $1
	`
)
