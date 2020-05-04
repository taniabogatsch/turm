package models

import (
	"strings"
	"turm/app"

	"github.com/revel/revel"
)

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

/*CourseListInfo holds only the most essential information about courses. */
type CourseListInfo struct {
	ID           int    `db:"id, primarykey, autoincrement"`
	Title        string `db:"title"`
	CreationDate string `db:"creationdate"`
	EMail        string `db:"email"` //e-mail address of either the creator or the editor
}

/*CourseList holds the most essential information about a list of courses. */
type CourseList []CourseListInfo

/*GetByUserID returns all courses according to the user type.  */
func (list *CourseList) GetByUserID(userID *int, userType string, active, expired bool) (err error) {

	//construct SQL
	stmtSelect := `
		SELECT c.id, c.title, u.email,
			TO_CHAR (c.creationdate AT TIME ZONE $1, 'YYYY-MM-DD HH24:MI') as creationdate
	`
	stmtWhere := `
			AND c.active = $3
			AND (current_timestamp >= c.expirationdate) = $4
		ORDER BY c.creationdate DESC
	`
	if !expired && !active {
		stmtWhere = `
				AND c.active = $3
				AND (
					(current_timestamp < c.expirationdate) = $4
					OR
					(current_timestamp >= c.expirationdate) = $4
				)
			ORDER BY c.creationdate DESC
		`
	}
	stmt := ``
	if userType == "creator" { //get all created courses
		stmt = `
		 	FROM course c, users u
		 	WHERE c.creator = u.id
		 		AND u.id = $2
		`
	} else { //get all edit/instruct privilege courses
		stmt = `
			FROM course c, users u, ` + userType + ` l
			WHERE c.id = l.courseid
				AND u.id = $2
				AND u.id = l.userid
		`
	}

	err = app.Db.Select(list, stmtSelect+stmt+stmtWhere, app.TimeZone, *userID, active, expired)
	if err != nil {
		modelsLog.Error("failed to get course list", "user ID", *userID,
			"userType", userType, "active", active, "expired", expired,
			"error", err.Error())
	}
	return
}

/*NewCourseParam holds all information about the different options to create a new course. */
type NewCourseParam struct {
	Title    string
	Option   Option
	CourseID int
	JSON     string
}

/*Validate NewCourseParam fields. */
func (param *NewCourseParam) Validate(v *revel.Validation) {

	param.Title = strings.TrimSpace(param.Title)
	v.Check(param.Title,
		revel.MinSize{3},
		revel.MaxSize{511},
	).MessageKey("validation.invalid.title")

	if param.Option < BLANK || param.Option > UPLOAD {
		v.ErrorKey("validation.invalid.option")
	} else if param.Option == DRAFT {
		v.Check(param.CourseID,
			revel.Required{},
			//TODO: user is only allowed to use drafts of courses that he created or of whom he was an editor
		).MessageKey("validation.invalid.courseID")
	} else if param.Option == UPLOAD {
		//TODO: validate json string
	}
	return
}
