package models

import (
	"path/filepath"
	"turm/app"

	"github.com/jmoiron/sqlx"
	"github.com/revel/revel"
)

/*EMailData holds all data that is rendered in the different e-mail templates. */
type EMailData struct {

	//used for salutation, user specific changes (new pw, role, activation, ...)
	User User

	//used for linking to the page
	URL string

	//used for enrollment
	CourseTitle string
	EventTitle  string
	CourseID    int
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
}

/*GetEMailSubjectBody assigns the template content to the e-mail body and sets the e-mail subject. */
func GetEMailSubjectBody(data *EMailData, language *string, subjectKey string,
	filename string, email *app.EMail, c *revel.Controller) (err error) {

	data.URL = app.Mailer.URL
	c.ViewArgs["data"] = data //set the data for parsing the e-mail body

	cLanguage := c.Session["currentLocale"].(string)
	c.ViewArgs["currentLocale"] = *language //set the preferred language for template parsing
	c.Request.Locale = *language

	email.Subject = c.Message(subjectKey) //set the e-mail subject
	email.ReplyTo = c.Message("email.no.reply", app.Mailer.EMail)

	//parse template / e-mail body
	filepath := filepath.Join("emails", filename+"_"+*language+".html")
	buf, err := revel.TemplateOutputArgs(filepath, c.ViewArgs)
	if err != nil {
		log.Error("failed to parse e-mail template", "filepath", filepath,
			"viewArgs", c.ViewArgs, "error", err.Error())
		return
	}
	email.Body = string(buf)

	c.ViewArgs["currentLocale"] = cLanguage //reset to original language
	c.Request.Locale = cLanguage
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

const (
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
