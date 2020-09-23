package models

type calendarEvent struct {
	id          int    `db:"id"`
	courseID    int    `db:"course_id"`
	title       string `db:"title"`
	annotations string `db:"annotations"`

	weekNr string //not in db

	days days
}

type days []day

type dayTemplate struct {
	id               int    `db:"id"`
	calendarEventID  int    `db:"calendar_event_id"`
	startTime        string `db:"start_time"`
	endTime          string `db:"end_time"`
	dayOfWeek        int    `db:"day_of_week"`
	active           bool   `db:"active"`
	deactiavtionDate string `db:"deactivation_date"`
}

type day struct {
	id            int    `db:"id"`
	dayTemplateID int    `db:"day_template_id"`
	calendarDate  string `db:"calendar_date"`
	intervall     int    `db:"intervall"`

	exeptions []exeptions
}

type slot struct {
	id        int    `db:"id"`
	userID    int    `db:"user_id"`
	dayID     int    `db:"day_id"`
	startTime string `db:"start_time"`
	endTime   string `db:"end_time"`
}

type exeptions []exeption

type exeption struct {
	id          int    `db:"id"`
	dayID       int    `db:"day_id"`
	startTime   string `db:"start_time"`
	endTime     string `db:"end_time"`
	annotations string `db:"annotations"`
}
