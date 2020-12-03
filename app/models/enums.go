package models

/*Salutation is a type for encoding different forms of address. */
type Salutation int

const (
	//NONE is for no form of address
	NONE Salutation = iota
	//MR is for Mr.
	MR
	//MS is for Ms.
	MS
)

func (s Salutation) String() string {
	return [...]string{"none", "mr", "ms"}[s]
}

/*Role is a type for encoding different user roles. */
type Role int

const (
	//USER is the default role without any extra privileges
	USER Role = iota
	//CREATOR allows the creation of courses
	CREATOR
	//ADMIN grants all privileges
	ADMIN
)

func (u Role) String() string {
	return [...]string{"user", "creator", "admin"}[u]
}

/*ScheduleEntryType is a type for encoding different schedule entries. */
type ScheduleEntryType int

const (
	//FREE is for no entry
	FREE ScheduleEntryType = iota
	//SLOT is for slots
	SLOT
	//EXCEPTION is for exceptions
	EXCEPTION
	//BLOCKED is for Timeslots between
	BLOCKED
)

func (s ScheduleEntryType) String() string {
	return [...]string{"free", "slot", "exception", "blocked"}[s]
}

/*Option encodes the different options to create a new course. */
type Option int

const (
	//BLANK is for empty courses
	BLANK Option = iota
	//DRAFT is for using existing courses
	DRAFT
	//UPLOAD is for uploading courses
	UPLOAD
)

func (op Option) String() string {
	return [...]string{"empty", "draft", "upload"}[op]
}

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
	//UNSUBSCRIBED users unsubscribed from an event
	UNSUBSCRIBED
)

func (status EnrollmentStatus) String() string {
	return [...]string{"enrolled", "on waitlist", "awaiting payment",
		"paid", "freed", "unsubscribed"}[status]
}

/*EnrollOption is a type for encoding different enrollment options. */
type EnrollOption int

const (
	//ENROLL is for normally enrolling in an event
	ENROLL EnrollOption = iota
	//UNSUBSCRIBE is for normally unsubscribing from an event
	UNSUBSCRIBE
	//NOENROLL disables the enrollment button
	NOENROLL
	//NOUNSUBSCRIBE disables the unsubscribe button
	NOUNSUBSCRIBE
	//ENROLLTOWAITLIST is for enrolling to the wait list
	ENROLLTOWAITLIST
	//UNSUBSCRIBEFROMWAITLIST is for unsubscribing from the wait list
	UNSUBSCRIBEFROMWAITLIST
)

func (s EnrollOption) String() string {
	return [...]string{"enroll", "unsubscribe", "noenroll", "nounsubscribe", "enrolltowaitlist"}[s]
}
