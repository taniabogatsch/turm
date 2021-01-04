package controllers

import (
	"time"
	"turm/app"

	"github.com/revel/revel"
)

//init initializes all interceptors.
func init() {

	//initialize general interceptor
	revel.InterceptFunc(general, revel.BEFORE, &revel.Controller{})

	//prevent unauthorized actions
	revel.InterceptMethod(Admin.auth, revel.BEFORE)
	revel.InterceptMethod(App.auth, revel.BEFORE)
	revel.InterceptMethod(Course.auth, revel.BEFORE)
	revel.InterceptMethod(Creator.auth, revel.BEFORE)
	revel.InterceptMethod(Edit.auth, revel.BEFORE)
	revel.InterceptMethod(EditEvent.auth, revel.BEFORE)
	revel.InterceptMethod(EditCalendarEvent.auth, revel.BEFORE)
	revel.InterceptMethod(EditMeeting.auth, revel.BEFORE)
	revel.InterceptMethod(Enrollment.auth, revel.BEFORE)
	revel.InterceptMethod(Manage.auth, revel.BEFORE)
	revel.InterceptMethod(Participants.auth, revel.BEFORE)
	revel.InterceptMethod(User.auth, revel.BEFORE)
}

func getTimestamp(str string, c *revel.Controller, valid bool, fieldID string) (t time.Time, err error) {

	if valid || fieldID != "unsubscribe_end" {

		loc, err := time.LoadLocation(app.TimeZone)
		if err != nil {
			c.Log.Error("failed to load location", "appTimeZone", app.TimeZone,
				"error", err.Error())
			return t, err
		}

		t, err = time.ParseInLocation("2006-01-02 15:04", str, loc)
		if err != nil {
			c.Log.Error("invalid timestamp", "str", str, "error", err.Error())
		}
	}

	return
}
