package models

import "turm/app"

/*UserDetails contains all data related to this user. */
type UserDetails struct {
	User User

	//all groups created by this user
	Groups Groups

	//all (former) enrollments of the user
	Enrollments       []Enrolled
	FormerEnrollments []Unsubscribed

	//all courses created by this user
	Courses []Course

	//all categories, faqs and news created by this user
	Categories []Category
	FAQs       []FAQ
	News       []NewsFeed
}

/*Get all user details. */
func (user *UserDetails) Get() (err error) {

	tx, err := app.Db.Beginx()
	if err != nil {
		modelsLog.Error("failed to begin tx", "error", err.Error())
		return err
	}

	//get user
	if err = user.User.Get(tx); err != nil {
		return
	}
	//get groups
	if err = user.Groups.GetByUser(&user.User.ID, tx); err != nil {
		return
	}

	//TODO: get Enrollments, FormerEnrollments, Courses, Categories, FAQs, News

	tx.Commit()
	return
}
