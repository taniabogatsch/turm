package models

import (
	"encoding/json"
	"strings"

	"github.com/revel/revel"
)

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
				log.Error("cannot unmarshal file", "file", string(param.JSON), "error", err.Error())
				v.ErrorKey("validation.invalid.file")
			}

			//case 1 or 2
			if jsonIntf["courseName"] != nil {

				if jsonIntf["enrolllimitevents"] != nil {
					//assert that enrolllimitevents is an integer
					switch jsonIntf["enrolllimitevents"].(type) {
					case bool:
						if jsonIntf["enrolllimitevents"] == true {
							jsonIntf["enrolllimitevents"] = 1
						} else {
							jsonIntf["enrolllimitevents"] = 0
						}
						//create an updated json to be unmarshalled into the course struct
						json, err := json.Marshal(jsonIntf)
						if err != nil {
							log.Error("cannot marshal file", "file", jsonIntf, "error", err.Error())
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
