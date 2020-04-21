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

/*Admin implements logic to CRUD dadta for admin functions. */
type Admin struct {
	*revel.Controller
}
