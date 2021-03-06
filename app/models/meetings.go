package models

import (
	"database/sql"
	"strings"
	"time"
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
	MeetingStart    time.Time       `db:"meeting_start"`
	MeetingEnd      time.Time       `db:"meeting_end"`

	//used to get the front end values
	MeetingStartTime string ``
	MeetingEndTime   string ``

	//used for pretty timestamp rendering
	MeetingStartStr string `db:"meeting_start_str"`
	MeetingEndStr   string `db:"meeting_end_str"`
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

	meeting.MeetingStartStr += " " + meeting.MeetingStartTime
	meeting.MeetingEndStr += " " + meeting.MeetingEndTime

	t, err := getTimestamp(meeting.MeetingStartStr)
	if err != nil {
		v.ErrorKey("validation.invalid.timestamp")
	}
	meeting.MeetingStart = t

	t, err = getTimestamp(meeting.MeetingEndStr)
	if err != nil {
		v.ErrorKey("validation.invalid.timestamp")
	}
	meeting.MeetingEnd = t

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
func (meeting *Meeting) NewBlank(conf *EditEMailConfig) (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	err = tx.Get(meeting, stmtInsertBlankMeeting, meeting.EventID, meeting.MeetingInterval)
	if err != nil {
		log.Error("failed to insert blank meeting", "meeting", meeting,
			"error", err.Error())
		tx.Rollback()
		return
	}

	if err = conf.Get(tx); err != nil {
		return
	}

	tx.Commit()
	return
}

/*Update a meeting. */
func (meeting *Meeting) Update(conf *EditEMailConfig) (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	if meeting.MeetingInterval == SINGLE {
		err = tx.Get(meeting, stmtUpdateSingleMeeting, meeting.Place,
			meeting.Annotation, meeting.MeetingStart, meeting.MeetingEnd, meeting.ID)
	} else {
		err = tx.Get(meeting, stmtUpdateWeeklyMeeting, meeting.Place,
			meeting.Annotation, meeting.MeetingStart, meeting.MeetingEnd,
			meeting.ID, meeting.WeekDay, meeting.MeetingInterval)
	}
	if err != nil {
		log.Error("failed to update meeting", "meeting", *meeting,
			"error", err.Error())
		tx.Rollback()
		return
	}

	if err = conf.Get(tx); err != nil {
		return
	}

	tx.Commit()
	return
}

/*Delete a meeting. */
func (meeting *Meeting) Delete(conf *EditEMailConfig) (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	err = deleteByID("id", "meetings", meeting.ID, tx)
	if err != nil {
		return
	}

	if err = conf.Get(tx); err != nil {
		return
	}

	tx.Commit()
	return
}

/*Duplicate a meeting. */
func (meeting *Meeting) Duplicate(conf *EditEMailConfig) (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	_, err = tx.Exec(stmtDuplicateMeeting, meeting.EventID, meeting.ID)
	if err != nil {
		log.Error("failed to duplicate meeting", "meeting",
			*meeting, "error", err.Error())
		tx.Rollback()
	}

	if err = conf.Get(tx); err != nil {
		return
	}

	tx.Commit()
	return
}

/*Meetings holds all meetings of an event. */
type Meetings []Meeting

/*Get all meetings of an event. */
func (meetings *Meetings) Get(tx *sqlx.Tx, eventID *int) (err error) {

	if tx == nil {
		err = app.Db.Select(meetings, stmtSelectMeetings, &eventID, app.TimeZone)
	} else {
		err = tx.Select(meetings, stmtSelectMeetings, &eventID, app.TimeZone)
	}

	if err != nil {
		log.Error("failed to get meetings of event", "event ID", *eventID, "error", err.Error())
		if tx != nil {
			tx.Rollback()
		}
	}
	return
}

/*Duplicate all meetings of an event. */
func (meetings *Meetings) Duplicate(tx *sqlx.Tx, eventIDNew, eventIDOld *int) (err error) {

	_, err = tx.Exec(stmtDuplicateMeetings, *eventIDNew, *eventIDOld)
	if err != nil {
		log.Error("failed to duplicate meetings", "eventIDNew",
			*eventIDNew, "eventIDOld", *eventIDOld, "error", err.Error())
		tx.Rollback()
	}

	return
}

/*Insert all meetings of an event. */
func (meetings *Meetings) Insert(tx *sqlx.Tx, eventID *int) (err error) {

	for _, meeting := range *meetings {
		_, err = tx.Exec(stmtInsertMeeting, meeting.Annotation, *eventID, meeting.MeetingEnd,
			meeting.MeetingInterval, meeting.MeetingStart, meeting.Place, meeting.WeekDay)
		if err != nil {
			log.Error("failed to insert meeting of event", "event ID", *eventID,
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
			meeting_start, meeting_end,
			TO_CHAR (meeting_start AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') as meeting_start_str,
			TO_CHAR (meeting_end AT TIME ZONE $2, 'YYYY-MM-DD HH24:MI') as meeting_end_str
		FROM meetings
		WHERE event_id = $1
		ORDER BY id ASC
	`

	stmtInsertBlankMeeting = `
		INSERT INTO meetings (event_id, meeting_interval)
		VALUES ($1, $2)
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
		SET place = $1, annotation = $2, meeting_start = $3, meeting_end = $4, weekday = $6,
			meeting_interval = $7
		WHERE id = $5
		RETURNING id
	`

	stmtDuplicateMeetings = `
		INSERT INTO meetings
			(annotation, event_id, meeting_end, meeting_start, meeting_interval,
				place, weekday)
		(
			SELECT
				annotation, $1 AS event_id, meeting_end, meeting_start, meeting_interval,
				place, weekday
			FROM meetings
			WHERE event_id = $2
		)
	`

	stmtDuplicateMeeting = `
		INSERT INTO meetings
			(annotation, event_id, meeting_end, meeting_start, meeting_interval,
				place, weekday)
		(
			SELECT
				annotation, $1 AS event_id, meeting_end, meeting_start, meeting_interval,
				place, weekday
			FROM meetings
			WHERE id = $2
		)
	`

	stmtInsertMeeting = `
		INSERT INTO meetings
			(annotation, event_id, meeting_end, meeting_interval, meeting_start, place, weekday)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	stmtGetCourseIDByMeeting = `
		SELECT e.course_id AS id
		FROM meetings m JOIN events e ON m.event_id = e.id
		WHERE m.id = $1
	`
)
