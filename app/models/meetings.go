package models

import (
	"database/sql"
	"strings"
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
	EventID         int             `db:"event_id"`
	MeetingInterval MeetingInterval `db:"meeting_interval"`
	WeekDay         sql.NullInt32   `db:"weekday"`
	Place           sql.NullString  `db:"place"`
	Annotation      sql.NullString  `db:"annotation"`
	MeetingStart    string          `db:"meeting_start"`
	MeetingEnd      string          `db:"meeting_end"`

	//used to get the front end values
	MeetingStartTime string ``
	MeetingEndTime   string ``
}

/*Validate meeting fields. */
func (meeting *Meeting) Validate(v *revel.Validation) {

	if meeting.MeetingInterval >= WEEKLY &&
		meeting.MeetingInterval <= ODD {

		if meeting.WeekDay.Int32 < int32(0) || meeting.WeekDay.Int32 > int32(6) {
			v.ErrorKey("validation.invalid-params")
		}
		meeting.WeekDay.Valid = true

	} else if meeting.MeetingInterval != SINGLE { //invalid meeting type
		v.ErrorKey("validation.invalid.params")
	}

	meeting.MeetingStart += " " + meeting.MeetingStartTime
	meeting.MeetingEnd += " " + meeting.MeetingEndTime

	v.Check(meeting.MeetingStart,
		IsTimestamp{},
	).MessageKey("validation.invalid.timestamp")

	v.Check(meeting.MeetingEnd,
		IsTimestamp{},
	).MessageKey("validation.invalid.timestamp")

	if meeting.Place.String != "" {

		meeting.Place.String = strings.TrimSpace(meeting.Place.String)
		v.Check(meeting.Place.String,
			revel.MinSize{3},
			revel.MaxSize{255},
		).MessageKey("validation.invalid.text.short")

		meeting.Place.Valid = true
	}

	if meeting.Annotation.String != "" {

		meeting.Annotation.String = strings.TrimSpace(meeting.Annotation.String)
		v.Check(meeting.Annotation.String,
			revel.MinSize{3},
			revel.MaxSize{255},
		).MessageKey("validation.invalid.text.short")

		meeting.Annotation.Valid = true
	}
}

/*NewBlank creates a new blank meeting. */
func (meeting *Meeting) NewBlank() (err error) {

	err = app.Db.Get(meeting, stmtInsertBlankMeeting, meeting.EventID, meeting.MeetingInterval)
	if err != nil {
		modelsLog.Error("failed to insert blank meeting", "meeting", meeting,
			"error", err.Error())
	}
	return
}

/*Update a meeting. */
func (meeting *Meeting) Update() (err error) {

	if meeting.MeetingInterval == SINGLE {
		err = app.Db.Get(meeting, stmtUpdateSingleMeeting, meeting.Place,
			meeting.Annotation, meeting.MeetingStart, meeting.MeetingEnd, meeting.ID)
	} else {
		err = app.Db.Get(meeting, stmtUpdateWeeklyMeeting, meeting.Place,
			meeting.Annotation, meeting.MeetingStart, meeting.MeetingEnd,
			meeting.ID, meeting.WeekDay)
	}
	if err != nil {
		modelsLog.Error("failed to update meeting", "meeting", meeting,
			"error", err.Error())
	}
	return
}

/*Delete a meeting. */
func (meeting *Meeting) Delete() (err error) {

	_, err = app.Db.Exec(stmtDeleteMeeting, meeting.ID)
	if err != nil {
		modelsLog.Error("failed to delete meeting", "meetingID", meeting.ID,
			"error", err.Error())
	}
	return
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

/*Duplicate all meetings of an event. */
func (meetings *Meetings) Duplicate(tx *sqlx.Tx, eventIDNew, eventIDOld *int) (err error) {

	_, err = tx.Exec(stmtDuplicateMeeting, *eventIDNew, *eventIDOld)
	if err != nil {
		modelsLog.Error("failed to duplicate event", "event ID new",
			*eventIDNew, "event ID old", *eventIDOld, "error", err.Error())
		tx.Rollback()
		return
	}
	return
}

/*Insert all meetings of an event. */
func (meetings *Meetings) Insert(tx *sqlx.Tx, eventID *int) (err error) {

	for _, meeting := range *meetings {
		_, err = tx.Exec(stmtInsertMeeting, meeting.Annotation, *eventID, meeting.MeetingEnd,
			meeting.MeetingInterval, meeting.MeetingStart, meeting.Place, meeting.WeekDay)
		if err != nil {
			modelsLog.Error("failed to insert meeting of event", "event ID", *eventID,
				"error", err.Error())
			tx.Rollback()
			return
		}
	}
	return
}

const (
	stmtSelectMeetings = `
		SELECT
			id, event_id, meeting_interval, weekday, place, annotation,
			TO_CHAR (meeting_start AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') as meeting_start,
			TO_CHAR (meeting_end AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') as meeting_end
		FROM meetings
		WHERE event_id = $1
		ORDER BY id ASC
	`

	stmtInsertBlankMeeting = `
		INSERT INTO meetings (
				event_id, meeting_start, meeting_end, meeting_interval
			)
		VALUES (
				$1, '2006-01-01 10:00', '2006-01-01 11:00', $2
		)
		RETURNING id
	`

	stmtUpdateSingleMeeting = `
		UPDATE meetings
		SET place = $1, annotation = $2, meeting_start = $3, meeting_end = $4
		WHERE id = $5
		RETURNING id
	`

	stmtUpdateWeeklyMeeting = `
		UPDATE meetings
		SET place = $1, annotation = $2, meeting_start = $3, meeting_end = $4, weekday = $6
		WHERE id = $5
		RETURNING id
	`

	stmtDeleteMeeting = `
		DELETE FROM meetings
		WHERE id = $1
	`

	stmtDuplicateMeeting = `
		INSERT INTO meetings
			(annotation, event_id, meeting_end, meeting_start, meeting_interval, place, weekday)
		(
			SELECT
				annotation, $1 AS event_id, meeting_end, meeting_start, meeting_interval, place, weekday
			FROM meetings
			WHERE event_id = $2
		)
	`

	stmtInsertMeeting = `
		INSERT INTO meetings
			(annotation, event_id, meeting_end, meeting_interval, meeting_start, place, weekday)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
)
