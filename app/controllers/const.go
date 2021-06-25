package controllers

//TODO: this should be in the models package?

const (
	//misc
	cTrue = "true"

	//help pages tables
	tabFAQCategory      = "faq_category"
	tabNewsFeedCategory = "news_feed_category"

	//course misc
	eventTypeNormal   = "normal"
	eventTypeCalendar = "calendar"

	//course tables
	tabWhitelists  = "whitelists"
	tabBlacklists  = "blacklists"
	tabInstructors = "instructors"
	tabEditors     = "editors"

	//course columns
	colTitle           = "title"
	colAnnotation      = "annotation"
	colHasComments     = "has_comments"
	colHasWaitlist     = "has_waitlist"
	colEnrollmentStart = "enrollment_start"
	colEnrollmentEnd   = "enrollment_end"
	colExpirationDate  = "expiration_date"
	colUnsubscribeEnd  = "unsubscribe_end"
	colVisible         = "visible"
	colOnlyLDAP        = "only_ldap"
	colFee             = "fee"
	colSubtitle        = "subtitle"
	colDescription     = "description"
	colSpeaker         = "speaker"
	colCustomEMail     = "custom_email"
)
