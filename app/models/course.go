package models

import (
	"database/sql"
	"math"
	"regexp"
	"strings"
	"time"
	"turm/app"

	"github.com/revel/revel"
)

/*Course is a model of the course table. */
type Course struct {
	ID                int             `db:"id, primarykey, autoincrement"`
	Title             string          `db:"title"`
	Creator           sql.NullInt32   `db:"creator"`
	Subtitle          sql.NullString  `db:"subtitle"`
	Visible           bool            `db:"visible"`
	Active            bool            `db:"active"`
	OnlyLDAP          bool            `db:"onlyldap"`
	CreationDate      string          `db:"creationdate"`
	Description       sql.NullString  `db:"description"`
	Speaker           sql.NullString  `db:"speaker"`
	Fee               sql.NullFloat64 `db:"fee"`
	CustomEMail       sql.NullString  `db:"customemail"`
	EnrollLimitEvents sql.NullInt32   `db:"enrolllimitevents"`
	EnrollmentStart   string          `db:"enrollmentstart"`
	EnrollmentEnd     string          `db:"enrollmentend"`
	UnsubscribeEnd    sql.NullString  `db:"unsubscribeend"`
	ExpirationDate    string          `db:"expirationdate"`
	ParentID          sql.NullInt32   `db:"parentid"`
	Events            Events          ``
	Editors           UserList        ``
	Instructors       UserList        ``
	Blacklist         UserList        ``
	Whitelist         UserList        ``
	Restrictions      []Restriction   ``

	//additional information required when displaying the course
	CreatorData User ``
	//path to the course entry in the groups tree
	Path Groups ``
}

/*Validate all Course fields. */
func (course *Course) Validate(v *revel.Validation) {

	v.Required(course.ID).
		MessageKey("validation.invalid.courseID")

	course.Title = strings.TrimSpace(course.Title)
	v.Check(course.Title,
		revel.MinSize{3},
		revel.MaxSize{511},
	).MessageKey("validation.invalid.title")

	//TODO: how to validate the creator?

	course.Subtitle.String = strings.TrimSpace(course.Subtitle.String)
	if course.Subtitle.String != "" {
		v.Check(course.Subtitle.String,
			revel.MinSize{3},
			revel.MaxSize{511},
		).MessageKey("validation.invalid.subtitle")
		course.Subtitle.Valid = true
	}

	if len(course.Restrictions) != 0 {
		v.Required(course.OnlyLDAP).
			MessageKey("validation.invalid.onlyLDAP")
	}

	if course.Description.String != "" {
		//TODO
		//v.Check(course.Description.String,
		//NoScript{}
		//).MessageKey("validation.invalid.description")
	}

	if course.Speaker.String != "" {
		//TODO
		//v.Check(course.Speaker.String,
		//NoScript{}
		//).MessageKey("validation.invalid.speaker")
	}

	if course.Fee.Float64 != 0.0 {
		course.Fee.Float64 = math.Round(course.Fee.Float64*100) / 100
		course.Fee.Valid = true
	}

	//TODO: all the other fields
	//TODO: must have entry in groups tree
}

/*Update the specified column in the course table. */
func (course *Course) Update(column string, value interface{}) (err error) {
	return updateByID(column, value, course.ID, "course", course)
}

/*Get all course data. */
func (course *Course) Get() (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		modelsLog.Error("failed to begin tx", "error", err.Error())
		return
	}

	err = tx.Get(course, stmtSelectCourse, course.ID, app.TimeZone)
	if err != nil {
		modelsLog.Error("failed to get course", "course ID", course.ID, "error", err.Error())
		tx.Rollback()
		return
	}
	if course.Fee.Valid {
		course.Fee.Float64 = math.Round(course.Fee.Float64*100) / 100
	}
	if err = course.Events.Get(tx, &course.ID); err != nil {
		return
	}
	if err = course.Editors.Get(tx, &course.ID, "editor"); err != nil {
		return
	}
	if err = course.Instructors.Get(tx, &course.ID, "instructor"); err != nil {
		return
	}
	if err = course.Blacklist.Get(tx, &course.ID, "blacklist"); err != nil {
		return
	}
	if err = course.Whitelist.Get(tx, &course.ID, "whitelist"); err != nil {
		return
	}

	//TODO: get restrictions

	//get more detailed creator data
	if course.Creator.Valid {
		course.CreatorData.ID = int(course.Creator.Int32)
		if err = course.CreatorData.GetBasicData(tx); err != nil {
			return
		}
	}

	//get the path of the course in the groups tree
	if err = course.Path.GetPath(&course.ID, tx); err != nil {
		return
	}

	tx.Commit()
	return
}

/*NewBlank creates a new blank course. */
func (course *Course) NewBlank(creatorID *int, title *string) (err error) {

	now := time.Now().Format(revel.TimeFormats[0])

	err = app.Db.Get(course, stmtInsertBlankCourse, now, *creatorID, *title)
	if err != nil {
		modelsLog.Error("failed to insert blank course", "now", now,
			"creator ID", *creatorID, "error", err.Error())
	}
	return
}

//FeePattern is the regular expression of accepted course fees
var FeePattern = regexp.MustCompile("^([0-9]{1,}(((,||.)[0-9]{1,2})||( )))?")

const (
	stmtSelectCourse = `
		SELECT
			id, title, creator, subtitle, visible, active, onlyldap, parentid,
			description, fee, customemail, enrolllimitevents, speaker,
			TO_CHAR (creationdate AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') as creationdate,
			TO_CHAR (enrollmentstart AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') as enrollmentstart,
			TO_CHAR (enrollmentend AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') as enrollmentend,
			TO_CHAR (unsubscribeend AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') as unsubscribeend,
			TO_CHAR (expirationdate AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') as expirationdate
		FROM course
		WHERE id = $1
	`

	stmtInsertBlankCourse = `
		INSERT INTO course (
				title, creator, visible, active, onlyldap, creationdate,
				enrollmentstart, enrollmentend, expirationdate
			)
		VALUES (
				$3, $2, false, false, false, $1, '2006-01-01',
				'2006-01-01', '2007-01-01'
		)
		RETURNING id
	`
)
