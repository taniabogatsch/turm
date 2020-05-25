package controllers

import "github.com/revel/revel"

/*App implements logic to CRUD general page data. */
type App struct {
	*revel.Controller
}

/*User implements logic to CRUD users. */
type User struct {
	*revel.Controller
}

/*Course implements logic to load courses. */
type Course struct {
	*revel.Controller
}

/*Edit implements logic to edit course data. */
type Edit struct {
	*revel.Controller
}

/*EditEvent implements logic to edit event data. */
type EditEvent struct {
	*revel.Controller
}

/*EditMeeting implements logic to edit meeting data. */
type EditMeeting struct {
	*revel.Controller
}

/*Creator implements logic to manage courses. */
type Creator struct {
	*revel.Controller
}

/*Manage implements the course management page. */
type Manage struct {
	*revel.Controller
}

/*Admin implements logic to CRUD data for admin functions. */
type Admin struct {
	*revel.Controller
}
