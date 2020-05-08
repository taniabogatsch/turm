package models

import (
	"encoding/json"
	"strings"

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

/*NewCourseParam holds all information about the different options to create a new course. */
type NewCourseParam struct {
	Title    string
	Option   Option
	CourseID int
	JSON     []byte
}

/*Validate NewCourseParam fields. */
func (param *NewCourseParam) Validate(v *revel.Validation, course *Course) {

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

		//validate file
		if json.Valid(param.JSON) {

			//NOTE: file might not be compatible with the current version, we must ensure backward compatibility
			//Unfortunately, there are three different file versions
			//1: old Turm2, enrolllimitevents is a boolean
			//2: old Turm2, enrolllimitevents is an integer
			//3: new Turm

			//unmarshal into interface to determine the file version
			var jsonIntf map[string]interface{}
			err := json.Unmarshal([]byte(param.JSON), &jsonIntf)
			if err != nil {
				modelsLog.Error("cannot unmarshal file", "file", string(param.JSON), "error", err.Error())
				v.ErrorKey("validation.invalid.file")
			}

			//case 1 or 2
			if jsonIntf["courseName"] != nil {

				if jsonIntf["enrolllimitevents"] != nil {
					//assert that enrolllimitevents is an integer
					switch jsonIntf["enrolllimitevents"].(type) {
					case bool:
						jsonIntf["enrolllimitevents"] = 0
						if jsonIntf["enrolllimitevents"] == true {
							jsonIntf["enrolllimitevents"] = 1
						}
						//create an updated json to be unmarshalled into the course struct
						json, err := json.Marshal(jsonIntf)
						if err != nil {
							modelsLog.Error("cannot marshal file", "file", jsonIntf, "error", err.Error())
							v.ErrorKey("validation.invalid.file")
						}
						param.JSON = json
					}
				}

				course.Load(true, &param.JSON)

			} else { //case 3
				course.Load(false, &param.JSON)
			}

		} else {
			v.ErrorKey("validation.invalid.json")
		}
	}
	return
}
