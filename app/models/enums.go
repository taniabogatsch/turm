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
