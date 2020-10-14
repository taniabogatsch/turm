package controllers

import (
	"github.com/revel/revel"
)

//init initializes all interceptors.
func init() {

	//initialize general interceptor
	revel.InterceptFunc(general, revel.BEFORE, &revel.Controller{})

	//prevent unauthorized actions
	revel.InterceptMethod(Admin.auth, revel.BEFORE)
	revel.InterceptMethod(App.auth, revel.BEFORE)
	revel.InterceptMethod(Creator.auth, revel.BEFORE)
	revel.InterceptMethod(Edit.auth, revel.BEFORE)
	revel.InterceptMethod(EditEvent.auth, revel.BEFORE)
	revel.InterceptMethod(EditCalendarEvent.auth, revel.BEFORE)
	revel.InterceptMethod(EditMeeting.auth, revel.BEFORE)
	revel.InterceptMethod(Enrollment.auth, revel.BEFORE)
	revel.InterceptMethod(Manage.auth, revel.BEFORE)
	revel.InterceptMethod(User.auth, revel.BEFORE)
}
