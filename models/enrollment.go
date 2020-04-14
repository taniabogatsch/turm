package models

import (
	"github.com/revel/revel"
)

/*EnrollmentStatus is a type for encoding the enrollment status. */
type EnrollmentStatus int

const (
	//ENROLLED is for users that enrolled in an event
	ENROLLED EnrollmentStatus = iota
	//ONWAITLIST is for users that are at the waitlist of an event
	ONWAITLIST
	//AWAITINGPAYMENT is for users that enrolled in an event but did not yet pay the fee of the course
	AWAITINGPAYMENT
	//PAID is for users that enrolled in an event and did pay the fee of the course
	PAID
	//FREED is for users that enrolled in an event and do not have to pay the fee of the course
	FREED
)

func (status EnrollmentStatus) String() string {
	return [...]string{"enrolled", "on waitlist", "awaiting payment", "paid", "freed"}[status]
}

/*Enrolled contains all directly enrollment status related values. */
type Enrolled struct {
	UserID           int              `db:"userid, primarykey"`
	EventID          int              `db:"eventid, primarykey"`
	Status           EnrollmentStatus `db:"status"`
	EMailTraffic     bool             `db:"emailtraffic"`
	TimeOfEnrollment string           `db:"timeofenrollment"`
}

//validateEnrolled validates the Enrolled struct fields.
func (enrolled *Enrolled) validateEnrolled(v *revel.Validation) {
	//TODO
}

/*Unsubscribed contains all fields of a user that unsubscribed from an event. */
type Unsubscribed struct {
	UserID  int `db:"userid, primarykey"`
	EventID int `db:"eventid, primarykey"`
}

//validateUnsubscribed validates the Enrolled struct fields.
func (unsubscribed *Unsubscribed) validateUnsubscribed(v *revel.Validation) {
	//TODO
}
