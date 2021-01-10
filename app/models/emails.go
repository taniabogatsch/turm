package models

import (
	"database/sql"
	"path/filepath"
	"strconv"
	"strings"
	"turm/app"

	"github.com/jmoiron/sqlx"
	"github.com/revel/revel"
)

/*EMailsData holds the data for multiple e-mails. */
type EMailsData []EMailData

/*EMailData holds all data that is rendered in the different e-mail templates. */
type EMailData struct {

	//used for salutation, user specific changes (new pw, role, activation, ...)
	User User

	//used for linking to the page
	URL string

	//used for enrollment
	CourseTitle string         `db:"course_title"`
	EventTitle  string         `db:"event_title"`
	CourseID    int            `db:"course_id"`
	CustomEMail sql.NullString `db:"custom_email"`

	//used for changing the enrollment status
	Status EnrollmentStatus

	//new course role type
	CourseRole string
	ViewMatrNr bool

	//used for slot enrollments
	Start  string `db:"start"`
	End    string `db:"end"`
	UserID int    `db:"user_id"`

	//used for notifying users about edits
	Field string

	//used for the custom enrollment e-mail
	CustomEMailData CustomEMailData
}

/*EditEMailConfig provides all information for sending edit notification e-mails. */
type EditEMailConfig struct {
	OptionUsers     int
	OptionEditors   int
	ID              int
	IsEvent         bool
	IsCalendarEvent bool

	Users              Users
	EditorsInstructors Users

	CourseTitle string `db:"course_title"`
	EventTitle  string `db:"event_title"`
	CourseID    int    `db:"course_id"`

	Field string
}

/*CustomEMailData contains all fields that can be used in the custom e-mail. */
type CustomEMailData struct {
	Salutation    Salutation     `db:"salutation"`
	Title         sql.NullString `db:"title"`
	NameAffix     sql.NullString `db:"name_affix"`
	AcademicTitle sql.NullString `db:"academic_title"`
	LastName      string         `db:"last_name"`
	FirstName     string         `db:"first_name"`
	CourseID      int            `db:"course_id"`
	CourseTitle   string         `db:"course_title"`
	EventTitle    string         `db:"event_title"`
	MeetingCount  int            `db:"meeting_count"`
	EMailCreator  string         `db:"email_creator"`
	Start         string         `db:"start"`
	End           string         `db:"end"`
	URL           string
}

/*GetEMailSubjectBody assigns the template content to the e-mail body and sets the e-mail subject. */
func GetEMailSubjectBody(data *EMailData, language *string, subjectKey string,
	filename string, email *app.EMail, c *revel.Controller) (err error) {

	data.URL = app.Mailer.URL

	email.Subject = c.Message(subjectKey) //set the e-mail subject
	email.ReplyTo = c.Message("email.no.reply", app.Mailer.EMail)

	//set the custom e-mail as the e-mail body
	if data.CustomEMail.Valid {

		data.CustomEMailData.URL = data.URL
		parseCustomEMail(&data.CustomEMail.String, &data.CustomEMailData, c)
		email.Body = app.HTMLToMimeFormat(&data.CustomEMail.String)

	} else { //parse the default e-mail template

		c.ViewArgs["data"] = data //set the data for parsing the e-mail body

		cLanguage := c.Session["currentLocale"].(string)
		c.ViewArgs["currentLocale"] = *language //set the preferred language for template parsing
		c.Request.Locale = *language

		//parse template / e-mail body
		filepath := filepath.Join("emails", filename+"_"+*language+".html")
		buf, err := revel.TemplateOutputArgs(filepath, c.ViewArgs)
		if err != nil {
			log.Error("failed to parse e-mail template", "filepath", filepath,
				"viewArgs", c.ViewArgs, "error", err.Error())
			return err
		}
		email.Body = string(buf)

		//reset to original language
		c.ViewArgs["currentLocale"] = cLanguage
		c.Request.Locale = cLanguage
	}

	return
}

/*Get all information for sending edit notification e-mails. */
func (conf *EditEMailConfig) Get(tx *sqlx.Tx) (err error) {

	if (conf.OptionUsers > 0 && conf.OptionUsers < 4) ||
		(conf.OptionEditors > 0 && conf.OptionEditors < 4) {

		//get to-be-notified users
		if conf.OptionUsers > 0 && conf.OptionUsers < 4 {

			//assemble the db statement
			stmt := `
				SELECT DISTINCT
					u.id, u.last_name, u.first_name, u.email, u.salutation, u.language, u.academic_title,
					u.title, u.name_affix, u.affiliations
				FROM users u JOIN enrolled e ON u.id = e.user_id`

			//courseID/eventID/calendarEventID
			if conf.IsEvent {
				stmt += ` WHERE e.event_id = $1`
			} else if conf.IsCalendarEvent {
				//TODO @Marco
			} else {
				stmt += ` JOIN events ev ON e.event_id = ev.id
					WHERE ev.course_id = $1`
			}

			//option
			if conf.OptionUsers == 2 {
				stmt += ` AND e.status != 1 /* not on wait list */`
			} else if conf.OptionUsers == 3 {
				stmt += ` AND e.status = 1 /* on wait list */`
			}

			//get users
			if err = tx.Select(&conf.Users, stmt, conf.ID); err != nil {
				log.Error("failed to get user for edit notification e-mail", "conf", *conf,
					"stmt", stmt, "error", err.Error())
				tx.Rollback()
				return
			}
		}

		//get to-be-notified editors/instructors
		if conf.OptionEditors > 0 && conf.OptionEditors < 4 {

			//assemble the db statement
			stmt := `
				SELECT
					u.id, u.last_name, u.first_name, u.email, u.salutation, u.language, u.academic_title,
					u.title, u.name_affix, u.affiliations
				FROM users u`

			stmtInstructors := stmt + ` JOIN instructors t ON u.id = t.user_id`
			stmtEditors := stmt + ` JOIN editors t ON u.id = t.user_id`
			stmtCreator := ``

			//courseID/eventID/calendarEventID
			if conf.IsEvent {

				stmtInstructors += ` JOIN events e ON t.course_id = e.course_id
					WHERE e.id = $1`
				stmtEditors += ` JOIN events e ON t.course_id = e.course_id
						WHERE e.id = $1`
				stmtCreator = ` UNION (
					SELECT u.id, u.last_name, u.first_name, u.email, u.salutation, u.language,
						u.academic_title, u.title, u.name_affix, u.affiliations
					FROM users u JOIN courses c ON u.id = c.creator
						JOIN events e ON c.id = e.course_id
					WHERE e.id = $1 )`

			} else if conf.IsCalendarEvent {
				//TODO @Marco
			} else {

				stmtInstructors += ` WHERE t.course_id = $1`
				stmtEditors += ` WHERE t.course_id = $1`
				stmtCreator = ` UNION (
					SELECT u.id, u.last_name, u.first_name, u.email, u.salutation, u.language,
						u.academic_title, u.title, u.name_affix, u.affiliations
					FROM users u JOIN courses c ON u.id = c.creator
					WHERE c.id = $1 )`
			}

			//option
			if conf.OptionEditors == 1 {
				stmt = `( ` + stmtInstructors + ` ) UNION ( ` + stmtEditors + ` )`
			} else if conf.OptionEditors == 2 {
				stmt = `( ` + stmtEditors + ` )`
			} else if conf.OptionEditors == 3 {
				stmt = `( ` + stmtInstructors + ` )`
			}

			//add creator
			stmt += stmtCreator

			//get creator/editors/instructors
			if err = tx.Select(&conf.EditorsInstructors, stmt, conf.ID); err != nil {
				log.Error("failed to get editors/instructors for edit notification e-mail", "conf", *conf,
					"stmt", stmt, "error", err.Error())
				tx.Rollback()
				return
			}
		}

		//get e-mail data
		if conf.IsEvent {
			if err = tx.Get(conf, stmtEventDataForEMail, conf.ID); err != nil {
				log.Error("failed to get event data for e-mail", "eventID", conf.ID,
					"error", err.Error())
				tx.Rollback()
				return
			}
		} else if conf.IsCalendarEvent {
			//TODO @Marco
		} else {
			if err = tx.Get(conf, stmtCourseDataForEMail, conf.ID); err != nil {
				log.Error("failed to get course data for e-mail", "courseID", conf.ID,
					"error", err.Error())
				tx.Rollback()
				return
			}
		}
	}

	return
}

func parseCustomEMail(content *string, data *CustomEMailData, c *revel.Controller) {

	//store the current language
	cLanguage := c.Session["currentLocale"].(string)

	//now parse for each language
	for _, language := range app.Languages {

		//set the language
		c.ViewArgs["currentLocale"] = language
		c.Request.Locale = language

		salutation := c.Message("user.salutation.none")
		if data.Salutation == MR {
			salutation = c.Message("user.salutation.mr")
		} else if data.Salutation == MS {
			salutation = c.Message("user.salutation.ms")
		}

		data.URL = data.URL + "/course/open?ID=" + strconv.Itoa(data.CourseID)

		*content = strings.ReplaceAll(*content, inBrackets(c.Message("user.salutation")), salutation)
		*content = strings.ReplaceAll(*content, inBrackets(c.Message("user.title")), data.Title.String)
		*content = strings.ReplaceAll(*content, inBrackets(c.Message("user.academic.title")), data.AcademicTitle.String)
		*content = strings.ReplaceAll(*content, inBrackets(c.Message("user.firstname")), data.FirstName)
		*content = strings.ReplaceAll(*content, inBrackets(c.Message("user.name.affix")), data.NameAffix.String)
		*content = strings.ReplaceAll(*content, inBrackets(c.Message("user.lastname")), data.LastName)
		*content = strings.ReplaceAll(*content, inBrackets(c.Message("course.title")), data.CourseTitle)
		*content = strings.ReplaceAll(*content, inBrackets(c.Message("event.title")), data.EventTitle)
		*content = strings.ReplaceAll(*content, inBrackets(c.Message("event.number.meetings")), strconv.Itoa(data.MeetingCount))
		*content = strings.ReplaceAll(*content, inBrackets(c.Message("course.creator.email")), data.EMailCreator)
		*content = strings.ReplaceAll(*content, inBrackets(c.Message("course.url")), data.URL)
		*content = strings.ReplaceAll(*content, inBrackets(c.Message("enroll.start.time")), data.Start)
		*content = strings.ReplaceAll(*content, inBrackets(c.Message("enroll.end.time")), data.End)
	}

	//reset to original language
	c.ViewArgs["currentLocale"] = cLanguage
	c.Request.Locale = cLanguage

	return
}

func (data *CustomEMailData) get(tx *sqlx.Tx, userID, courseID, eventID, slotID int) (err error) {

	if slotID != 0 {
		err = tx.Get(data, stmtGetCustomEMailDataSlot, userID, courseID, eventID, slotID, app.TimeZone)
	} else {
		err = tx.Get(data, stmtGetCustomEMailDataEvent, userID, courseID, eventID)
	}

	if err != nil {
		log.Error("failed to get custom e-mail data by event", "userID", userID,
			"courseID", courseID, "eventID", eventID, "slotID", slotID, "error", err.Error())
		tx.Rollback()
		return
	}

	return
}

func inBrackets(str string) string {

	return "[[" + str + "]]"
}

const (
	stmtGetCustomEMailDataEvent = `
		SELECT u.salutation, u.title, u.name_affix, u.academic_title, u.last_name,
			u.first_name, c.id AS course_id, c.title AS course_title, e.title AS event_title,
			COUNT(m.id) AS meeting_count, uc.email AS email_creator
		FROM users u, courses c
		 	JOIN users uc ON c.creator = uc.id
			JOIN events e ON c.id = e.course_id
			LEFT OUTER JOIN meetings m ON e.id = m.event_id
		WHERE u.id = $1
			AND c.id = $2
			AND e.id = $3
		GROUP BY u.salutation, u.title, u.name_affix, u.academic_title, u.last_name,
			u.first_name, c.id, c.title, e.title, uc.email
	`

	stmtGetCustomEMailDataSlot = `
		SELECT u.salutation, u.title, u.name_affix, u.academic_title, u.last_name,
			u.first_name, c.id AS course_id, c.title AS course_title, e.title AS event_title,
			uc.email AS email_creator,
			TO_CHAR (s.start_time AT TIME ZONE $5, 'YYYY-MM-DD HH24:MI') AS start,
			TO_CHAR (s.end_time AT TIME ZONE $5, 'YYYY-MM-DD HH24:MI') AS end
		FROM users u, courses c
			JOIN users uc ON c.creator = uc.id
			JOIN calendar_events e ON c.id = e.course_id
			JOIN day_templates d ON d.calendar_event_id = e.id
			JOIN slots s ON s.day_tmpl_id = d.id
		WHERE u.id = $1
			AND c.id = $2
			AND e.id = $3
			AND s.id = $4
	`

	stmtEventDataForEMail = `
		SELECT e.title AS event_title, c.title AS course_title,
			c.id AS course_id
		FROM events e JOIN courses c ON e.course_id = c.id
		WHERE e.id = $1
	`

	stmtCourseDataForEMail = `
		SELECT id AS course_id, title AS course_title
		FROM courses
		WHERE id = $1
	`
)
