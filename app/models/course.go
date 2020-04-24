package models

import (
	"database/sql"
	"time"
	"turm/app"

	"github.com/revel/revel"
)

/*Course contains all directly course related values. */
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
	Events            Events          ``
	Editors           []UserList      ``
	Instructors       []UserList      ``
	Blacklist         []UserList      ``
	Whitelist         []UserList      ``
	Restrictions      []Restriction   ``

	//additional information required when displaying the course
	CreatorData User ``
}

/*Validate validates the Course struct fields. */
func (course *Course) Validate(v *revel.Validation) {
	//TODO
}

/*Get all course data. */
func (course *Course) Get() (err error) {

	selectCourse := `
		SELECT
			id, title, creator, subtitle, visible, active, onlyldap,
			description, fee, customemail, enrolllimitevents, speaker,
			TO_CHAR (creationdate AT TIME ZONE $2, 'DD.MM.YYYY HH24:MI') as creationdate,
			TO_CHAR (enrollmentstart AT TIME ZONE $2, 'DD.MM.YYYY HH24:MI') as enrollmentstart,
			TO_CHAR (enrollmentend AT TIME ZONE $2, 'DD.MM.YYYY HH24:MI') as enrollmentend,
			TO_CHAR (unsubscribeend AT TIME ZONE $2, 'DD.MM.YYYY HH24:MI') as unsubscribeend,
			TO_CHAR (expirationdate AT TIME ZONE $2, 'DD.MM.YYYY HH24:MI') as expirationdate
		FROM course
		WHERE id = $1
	`

	tx, err := app.Db.Beginx()
	if err != nil {
		modelsLog.Error("failed to begin tx", "error", err.Error())
		return
	}

	err = tx.Get(course, selectCourse, course.ID, app.TimeZone)
	if err != nil {
		modelsLog.Error("failed to get course", "course ID", course.ID, "error", err.Error())
		tx.Rollback()
		return
	}

	if err = course.Events.Get(tx, &course.ID); err != nil {
		return
	}

	//TODO: get editors
	//TODO: get instructors
	//TODO: get blacklist
	//TODO: get whitelist
	//TODO: get restrictions

	//get more detailed creator data
	if course.Creator.Valid {
		course.CreatorData.ID = int(course.Creator.Int32)
		if err = course.CreatorData.GetBasicData(tx); err != nil {
			return
		}
	}

	tx.Commit()
	return
}

/*NewBlank creates a new blank course. */
func (course *Course) NewBlank(creatorID *int, title *string) (err error) {

	insertBlankCourse := `
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
	now := time.Now().Format(revel.TimeFormats[0])

	err = app.Db.Get(course, insertBlankCourse, now, *creatorID, *title)
	if err != nil {
		modelsLog.Error("failed to insert blank course", "now", now,
			"creator ID", *creatorID, "error", err.Error())
	}
	return
}
