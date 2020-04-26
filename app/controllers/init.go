package controllers

import (
	"github.com/revel/revel"
)

//init initializes all interceptors.
func init() {

	//initialize general interceptor
	revel.InterceptFunc(general, revel.BEFORE, &revel.Controller{})

	//prevent unauthorized actions
	revel.InterceptMethod(Admin.authAdmin, revel.BEFORE)
	revel.InterceptMethod(App.authApp, revel.BEFORE)
	revel.InterceptMethod(Creator.authCreator, revel.BEFORE)
	revel.InterceptMethod(EditCourse.authEditCourse, revel.BEFORE)
	revel.InterceptMethod(User.authUser, revel.BEFORE)
}
