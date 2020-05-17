package models

import "turm/app"

/*UserDetails holds detailed data related to a user. */
type UserDetails struct {
	User User

	//all groups created by this user
	Groups Groups

	//all (former) enrollments of the user
	Enrollments       []Enrolled
	FormerEnrollments []Unsubscribed

	//all courses in which the user was directly involved
	CreatedCourses []Course
	EditorOf       []Course
	InstructorOf   []Course

	//all courses of which the user was on the whitelist/blacklist
	OnWhitelist []Course
	OnBlacklist []Course

	//all categories, faqs and news created by this user
	Categories []Category
	FAQs       []HelpPageEntries
	News       []HelpPageEntries
}

/*Get all user details. */
func (user *UserDetails) Get() (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	//get user
	if err = user.User.Get(tx); err != nil {
		return
	}
	//get groups
	if err = user.Groups.SelectByUser(&user.User.ID, tx); err != nil {
		return
	}

	//TODO: get Enrollments, FormerEnrollments, Courses, Categories, FAQs, News

	tx.Commit()
	return
}
