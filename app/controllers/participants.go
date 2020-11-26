package controllers

import (
	"encoding/csv"
	"os"
	"strconv"
	"strings"
	"time"
	"turm/app"
	"turm/app/models"

	"github.com/revel/revel"
)

//TODO: interceptor and roles!
//TODO: interceptor: event must be part of course

/*Open a course for user management. */
func (c Participants) Open(ID, eventID int) revel.Result {

	c.Log.Debug("open course for user management", "ID", ID)

	//TODO: the interceptor assures that the course ID is valid

	//get the user ID, because not all editors/instructors are allowed to
	//see matriculation numbers
	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		renderQuietError(errTypeConv, err, c.Controller)
		return c.Render()
	}

	//get the participants
	participants := models.Participants{ID: ID}
	if err := participants.Get(userID); err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	//only set these after the course is loaded
	c.Session["callPath"] = c.Request.URL.String()
	c.Session["currPath"] = c.Request.URL.String()
	c.ViewArgs["tabName"] = c.Message("pcpts.tab")

	if eventID == 0 {
		if len(participants.Lists) > 0 {
			eventID = participants.Lists[0].ID
		}
	}

	return c.Render(participants, eventID)
}

/*Download a list of participants. */
func (c Participants) Download(ID int, conf models.ListConf) revel.Result {

	c.Log.Debug("download list of participants", "ID", ID, "conf", conf)

	//get the user ID, because not all editors/instructors are allowed to
	//see matriculation numbers
	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		return flashError(
			errTypeConv, err, "", c.Controller, "")
	}

	//get the participants
	participants := models.Participants{ID: ID}
	if err := participants.Get(userID); err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
	}

	//create the file and get the filepath
	filepath, err := createCSV(c.Controller, &participants, &conf)
	if err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
	}

	return c.RenderFileName(filepath, revel.Attachment)
}

/*EMail a list of participants. */
func (c Participants) EMail(ID int, conf models.ListConf) revel.Result {

	c.Log.Debug("send an e-mail to a list of participants", "ID", ID, "conf", conf)

	//get the user ID, because not all editors/instructors are allowed to
	//see matriculation numbers
	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		return flashError(
			errTypeConv, err, "", c.Controller, "")
	}

	//get the participants
	participants := models.Participants{ID: ID}
	if err := participants.Get(userID); err != nil {
		return flashError(
			errDB, err, "", c.Controller, "")
	}

	//get all e-mail recipients
	emails := make(map[string]bool)
	for _, event := range participants.Lists {

		if conf.AllEvents || containsEvent(conf.EventIDs, event.ID) {

			//participants
			if conf.Participants && len(event.Participants) != 0 {
				for _, user := range event.Participants {
					_, exists := emails[user.EMail]
					if !exists {
						emails[user.EMail] = true
					}
				}
			}

			//wait list
			if conf.WaitList && len(event.Waitlist) != 0 {
				for _, user := range event.Waitlist {
					_, exists := emails[user.EMail]
					if !exists {
						emails[user.EMail] = true
					}
				}
			}

			//unsubscribed
			if conf.Unsubscribed && len(event.Unsubscribed) != 0 {
				for _, user := range event.Unsubscribed {
					_, exists := emails[user.EMail]
					if !exists {
						emails[user.EMail] = true
					}
				}
			}

		}
	}

	//send e-mails
	for email := range emails {
		email := app.EMail{
			Recipient: email,
			Subject:   conf.Subject,
			ReplyTo:   participants.UserEMail,
			Body:      app.HTMLToMimeFormat(&conf.Content),
		}
		app.EMailQueue <- email
	}

	c.Flash.Success(c.Message("email.send.success", len(emails)))
	return c.Redirect(Participants.Open, ID)
}

/*SearchUser renders search results for a search value. */
func (c Participants) SearchUser(ID, eventID int, value string) revel.Result {

	c.Log.Debug("search users", "ID", ID, "eventID", eventID, "value", value)

	value = strings.TrimSpace(value)
	c.Validation.Check(value,
		revel.MinSize{3},
		revel.MaxSize{127},
	).MessageKey("validation.invalid.searchValue")

	if c.Validation.HasErrors() {
		c.Validation.Keep()
		return c.Render()
	}

	//get the user ID, because not all editors/instructors are allowed to
	//see matriculation numbers
	userID, err := getIntFromSession(c.Controller, "userID")
	if err != nil {
		renderQuietError(errTypeConv, err, c.Controller)
	}

	var entries models.Entries
	hasWaitlist, err := entries.Search(&ID, &eventID, &userID, &value)
	if err != nil {
		renderQuietError(errDB, err, c.Controller)
		return c.Render()
	}

	return c.Render(entries, ID, eventID, hasWaitlist)
}

/*Enroll a user without validating enrollment constraints. */
func (c Participants) Enroll(ID, eventID, userID int) revel.Result {

	c.Log.Debug("enroll user without constraints", "ID", ID,
		"eventID", eventID, "userID", userID)

	//enroll user
	enrolled := models.Enrolled{EventID: eventID, UserID: userID}
	data, err := enrolled.Enroll(&ID, c.Validation)
	if err != nil {
		return flashError(errDB, err, "", c.Controller, "")
	} else if c.Validation.HasErrors() {
		return flashError(
			errValidation, nil, "", c.Controller, "")
	}

	//send e-mail to the user
	err = sendEMail(c.Controller, &data,
		"email.subject.manual.enroll",
		"manualEnroll")

	if err != nil {
		return flashError(
			errEMail, err, "", c.Controller, data.User.EMail)
	}

	c.Flash.Success(c.Message("enroll.manual.success"))
	return c.Redirect(Participants.Open, ID, eventID)
}

/*Unsubscribe a user from an event. */
func (c Participants) Unsubscribe(ID, eventID, userID int) revel.Result {

	c.Log.Debug("unsubscribe user from an event", "ID", ID,
		"eventID", eventID, "userID", userID)

	//unsubscribe user
	enrolled := models.Enrolled{EventID: eventID, UserID: userID}
	data, users, err := enrolled.Unsubscribe(&ID, c.Validation)
	if err != nil {
		return flashError(errDB, err, "", c.Controller, "")
	} else if c.Validation.HasErrors() {
		return flashError(
			errValidation, nil, "", c.Controller, "")
	}

	//send e-mail to the user
	err = sendEMail(c.Controller, &data,
		"email.subject.manual.unsubscribed",
		"manualUnsub")

	if err != nil {
		return flashError(
			errEMail, err, "", c.Controller, data.User.EMail)
	}

	//send e-mail to each auto enrolled user
	for _, user := range users {
		mailData := models.EMailData{
			User:        user,
			CourseTitle: data.CourseTitle,
			EventTitle:  data.EventTitle,
			CourseID:    data.CourseID,
		}
		err = sendEMail(c.Controller, &mailData,
			"email.subject.from.wait.list",
			"fromWaitlist")
		if err != nil {
			return flashError(
				errEMail, err, "", c.Controller, mailData.User.EMail)
		}
	}

	c.Flash.Success(c.Message("enroll.manual.unsubscribe.success"))
	return c.Redirect(Participants.Open, ID, eventID)
}

/*Waitlist puts a user at the wait list of an event without validating
enrollment constraints. */
func (c Participants) Waitlist(ID, eventID, userID int) revel.Result {

	c.Log.Debug("put user at wait list without validating constraints", "ID", ID,
		"eventID", eventID, "userID", userID)

	//enroll user to wait list
	enrolled := models.Enrolled{EventID: eventID, UserID: userID}
	data, users, err := enrolled.Waitlist(&ID, c.Validation)
	if err != nil {
		return flashError(errDB, err, "", c.Controller, "")
	} else if c.Validation.HasErrors() {
		return flashError(
			errValidation, nil, "", c.Controller, "")
	}

	//send e-mail to the user
	err = sendEMail(c.Controller, &data,
		"email.subject.manual.wait.list",
		"manualWaitlist")

	if err != nil {
		return flashError(
			errEMail, err, "", c.Controller, data.User.EMail)
	}

	//send e-mail to each auto enrolled user
	for _, user := range users {
		mailData := models.EMailData{
			User:        user,
			CourseTitle: data.CourseTitle,
			EventTitle:  data.EventTitle,
			CourseID:    data.CourseID,
		}
		err = sendEMail(c.Controller, &mailData,
			"email.subject.from.wait.list",
			"fromWaitlist")
		if err != nil {
			return flashError(
				errEMail, err, "", c.Controller, mailData.User.EMail)
		}
	}

	c.Flash.Success(c.Message("enroll.manual.to.wait.list.success"))
	return c.Redirect(Participants.Open, ID, eventID)
}

func createCSV(c *revel.Controller, participants *models.Participants,
	conf *models.ListConf) (filepath string, err error) {

	//data that will be written to the csv-file
	var data [][]string

	//get date and time
	year, month, day := time.Now().Date()
	hour, minute, _ := time.Now().Clock()
	date := strconv.Itoa(year) + "-" + strconv.Itoa(int(month)) + "-" + strconv.Itoa(day)
	time := strconv.Itoa(hour) + ":" + strconv.Itoa(minute)

	//no custom filename set
	if conf.Filename == "" {
		conf.Filename = date + "_" + participants.Title
	}
	filepath = "/tmp/" + conf.Filename + ".csv"

	//course ID, title and extraction time
	row := []string{c.Message("course.title") + ": " + participants.Title}
	data = append(data, row)
	row = []string{c.Message("course.ID") + ": " + strconv.Itoa(participants.ID)}
	data = append(data, row)
	row = []string{c.Message("pcpts.download.extraction.time") + ": " + date + " " + time}
	data = append(data, row)

	//first row with headings
	row = []string{}
	data = append(data, row)
	row = []string{}
	row = append(row,
		c.Message("event.ID"),
		c.Message("event.title"),
		c.Message("user.salutation"),
		c.Message("user.academic.title"),
		c.Message("user.title"),
		c.Message("user.firstname"),
		c.Message("user.name.affix"),
		c.Message("user.lastname"),
		c.Message("user.email"),
		c.Message("user.language"),
		c.Message("user.matr.nr"),
		c.Message("user.affiliation"),
		c.Message("user.degree"),
		c.Message("user.course.of.studies"),
		c.Message("user.semester"),
		c.Message("enroll.time"),
		c.Message("enroll.status"))
	data = append(data, row)
	row = []string{}
	data = append(data, row)

	for _, event := range participants.Lists {

		if conf.AllEvents || containsEvent(conf.EventIDs, event.ID) {

			//participants
			if conf.Participants && len(event.Participants) != 0 {
				row = []string{}
				data = append(data, row)
				appendList(&data, event.Participants, c, event.ID, event.Title)
			}

			//wait list
			if conf.WaitList && len(event.Waitlist) != 0 {
				row = []string{}
				data = append(data, row)
				appendList(&data, event.Waitlist, c, event.ID, event.Title)
			}

			//unsubscribed
			if conf.Unsubscribed && len(event.Unsubscribed) != 0 {
				row = []string{}
				data = append(data, row)
				appendList(&data, event.Unsubscribed, c, event.ID, event.Title)
			}

		}
	}

	//now create the actual file
	file, err := os.Create(filepath)
	if err != nil {
		c.Log.Error("failed to create file for download", "filepath", filepath,
			"error", err.Error())
		return
	}
	defer file.Close() //close file at the end of the function

	//write data to file
	writer := csv.NewWriter(file)
	writer.Comma = ';'
	if conf.UseComma {
		writer.Comma = ','
	}
	defer writer.Flush()

	for _, line := range data {
		err = writer.Write(line)
		if err != nil {
			c.Log.Error("failed to write data to csv", "line", line,
				"error", err.Error())
			return
		}
	}

	return
}

func appendList(data *[][]string, list models.Entries, c *revel.Controller,
	ID int, title string) {

	for _, user := range list {

		row := []string{}

		salutation := c.Message("user.salutation.ms")
		if user.Salutation == models.NONE {
			salutation = c.Message("user.salutation.none")
		} else if user.Salutation == models.MR {
			salutation = c.Message("user.salutation.mr")
		}

		//matriculation number
		matrNr := c.Message("user.no.matr.nr")
		if user.MatrNr.Valid {
			if user.MatrNr.Int32 != 12345 {
				matrNr = strconv.Itoa(int(user.MatrNr.Int32))
			} else {
				matrNr = c.Message("user.matr.nr.not.visible")
			}
		}

		//convert array of affiliations to string
		affiliations := stringFromSlice(user.Affiliations.Affiliations)

		degrees, studies, semesters := "", "", ""
		for _, study := range user.Studies {
			degrees = appendValueToString(degrees, study.Degree)
			studies = appendValueToString(studies, study.CourseOfStudies)
			semesters = appendValueToString(semesters, strconv.Itoa(study.Semester))
		}

		enrollStatus := c.Message("enroll.status.enrolled")
		switch user.Status {
		case models.ONWAITLIST:
			enrollStatus = c.Message("enroll.status.on.wait.list")
		case models.AWAITINGPAYMENT:
			enrollStatus = c.Message("enroll.status.awaiting.payment")
		case models.PAID:
			enrollStatus = c.Message("enroll.status.paid")
		case models.FREED:
			enrollStatus = c.Message("enroll.status.freed")
		case models.UNSUBSCRIBED:
			enrollStatus = c.Message("enroll.status.unsubscribed")
		}

		row = append(row,
			strconv.Itoa(ID),
			title,
			salutation,
			user.AcademicTitle.String,
			user.Title.String,
			user.FirstName,
			user.NameAffix.String,
			user.LastName,
			user.EMail,
			user.Language.String,
			matrNr,
			affiliations,
			degrees,
			studies,
			semesters,
			user.TimeOfEnrollment,
			enrollStatus)

		//and put them in the csv data array
		*data = append(*data, row)
	}
}

func containsEvent(IDs []int, ID int) bool {

	for _, value := range IDs {
		if value == ID {
			return true
		}
	}
	return false
}

func stringFromSlice(slice []string) (str string) {

	for _, value := range slice {
		str = appendValueToString(str, value)
	}
	return str
}

func appendValueToString(str, value string) string {

	if str == "" {
		return value
	}
	return str + ", " + value
}
