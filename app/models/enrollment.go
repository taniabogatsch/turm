package models

import (
	"github.com/revel/revel"
)

/*EnrollmentStatus is a type for encoding the enrollment status. */
type EnrollmentStatus int

const (
	//ENROLLED users enrolled in an event
	ENROLLED EnrollmentStatus = iota
	//ONWAITLIST users are at the waitlist of an event
	ONWAITLIST
	//AWAITINGPAYMENT users enrolled in an event but did not yet pay the fee of the course
	AWAITINGPAYMENT
	//PAID users enrolled in an event and did pay the fee of the course
	PAID
	//FREED users enrolled in an event and do not have to pay the fee of the course
	FREED
)

func (status EnrollmentStatus) String() string {
	return [...]string{"enrolled", "on waitlist", "awaiting payment", "paid", "freed"}[status]
}

/*Enrolled is a model of the enrolled table. */
type Enrolled struct {
	UserID           int              `db:"user_id, primarykey"`
	EventID          int              `db:"event_id, primarykey"`
	Status           EnrollmentStatus `db:"status"`
	EMailTraffic     bool             `db:"email_traffic"`
	TimeOfEnrollment string           `db:"time_of_enrollment"`
}

/*Validate Enrolled fields. */
func (enrolled *Enrolled) Validate(v *revel.Validation) {
	//TODO
}

/*Unsubscribed is a model of the unsubscribed table. */
type Unsubscribed struct {
	UserID  int `db:"user_id, primarykey"`
	EventID int `db:"event_id, primarykey"`
}

/*Validate Unsubscribed fields. */
func (unsubscribed *Unsubscribed) Validate(v *revel.Validation) {
	//TODO
}
