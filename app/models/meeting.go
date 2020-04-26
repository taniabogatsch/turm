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
	//SINGLE meetings happen once
	SINGLE MeetingInterval = iota
	//WEEKLY meetings happen every week
	WEEKLY
	//EVEN meetings happen in even weeks
	EVEN
	//ODD meetings happen in odd weeks
	ODD
)

func (interval MeetingInterval) String() string {
	return [...]string{"single", "weekly", "even", "odd"}[interval]
}

/*Meeting is a model of the meeting table. */
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

/*Validate meeting fields. */
func (event *Meeting) Validate(v *revel.Validation) {
	//TODO
}

/*Meetings holds all meetings of an event. */
type Meetings []Meeting

/*Get all meetings of an event. */
func (meetings *Meetings) Get(tx *sqlx.Tx, eventID *int) (err error) {

	err = tx.Select(meetings, stmtSelectMeetings, &eventID, app.TimeZone)
	if err != nil {
		modelsLog.Error("failed to get meetings of event", "event ID", *eventID, "error", err.Error())
		tx.Rollback()
	}
	return
}

const (
	stmtSelectMeetings = `
		SELECT
			id, eventid, meetinginterval, weekday, place, annotation,
			TO_CHAR (meetingstart AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') as meetingstart,
			TO_CHAR (meetingend AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') as meetingend
		FROM meeting
		WHERE eventid = $1
		ORDER BY id ASC
	`
)
