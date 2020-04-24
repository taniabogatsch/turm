package models

import (
	"database/sql"
	"turm/app"

	"github.com/jmoiron/sqlx"
	"github.com/revel/revel"
)

/*MeetingInterval is a type for encoding different types of meetings. */
type MeetingInterval int

const (
	//SINGLE is for meetings that happen once
	SINGLE MeetingInterval = iota
	//WEEKLY is for meetings that happen every week
	WEEKLY
	//EVEN is for meetings that happen in even weeks
	EVEN
	//ODD is for meetings that happen in odd weeks
	ODD
)

func (interval MeetingInterval) String() string {
	return [...]string{"single", "weekly", "even", "odd"}[interval]
}

/*Meeting contains all directly meeting related values. */
type Meeting struct {
	ID              int             `db:"id, primarykey, autoincrement"`
	EventID         int             `db:"eventid"`
	MeetingInterval MeetingInterval `db:"meetinginterval"`
	WeekDay         sql.NullInt32   `db:"weekday"`
	Place           sql.NullString  `db:"place"`
	Annotation      sql.NullString  `db:"annotation"`
	MeetingStart    string          `db:"meetingstart"`
	MeetingEnd      string          `db:"meetingend"`
}

/*Validate validates the Meeting struct fields. */
func (event *Meeting) Validate(v *revel.Validation) {
	//TODO
}

/*Meetings contains all meetings of an event. */
type Meetings []Meeting

/*Get all meetings of an event. */
func (meetings *Meetings) Get(tx *sqlx.Tx, eventID *int) (err error) {

	selectMeetings := `
		SELECT
			id, eventid, meetinginterval, weekday, place, annotation,
			TO_CHAR (meetingstart AT TIME ZONE $2, 'DD.MM.YYYY HH24:MI') as meetingstart,
			TO_CHAR (meetingend AT TIME ZONE $2, 'DD.MM.YYYY HH24:MI') as meetingend
		FROM meeting
		WHERE eventid = $1
		ORDER BY id ASC
	`
	err = tx.Select(meetings, selectMeetings, &eventID, app.TimeZone)
	if err != nil {
		modelsLog.Error("failed to get meetings of event", "event ID", *eventID, "error", err.Error())
		tx.Rollback()
	}
	return
}
