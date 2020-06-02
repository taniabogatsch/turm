package app

import (
	"bytes"
	"encoding/csv"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/revel/revel"
)

var (
	//cloud holds all DB backup cloud data
	cloud cloudConn

	//enrollFilePath holds the path to the enrollment data file
	enrollFilePath string

	//jobSchedules holds the time of each scheduled job
	jobSchedules map[string]string
)

//cloudConn contains all cloud connection and upload fields
type cloudConn struct {
	Path     string
	Address  string
	Folder   string
	User     string
	Password string
}

//backupDB is a struct that implements the functionality to back up the DB
type backupDB struct{}

/*Run executes the DB backup job. */
func (e backupDB) Run() {

	if !revel.DevMode {
		if err := backup(); err != nil {
			SendErrorNote()
		}

	} else {
		revel.AppLog.Warn("not in production mode, skip backing up DB...")
	}
}

//back up the DB locally and upload it to the specified upload location
func backup() (err error) {

	revel.AppLog.Warn("start creating DB backup...")

	//connection
	connStr := "--dbname=postgresql://" + dbData.User + ":" + dbData.Password +
		"@" + dbData.Host + ":" + dbData.Port + "/" + dbData.Name

	//file config
	now := time.Now().Format("2006-01-02_15:04:05")
	filename := now + "_DBdump.sql"
	fpath := filepath.Join(cloud.Path, filename)

	//create a local backup
	out, err := exec.Command("pg_dump", "--no-owner", connStr, "-f", fpath).CombinedOutput()
	if err != nil {
		revel.AppLog.Error("failed to create local backup", "connStr", connStr, "filepath",
			fpath, "err", err.Error(), "out", string(out))
		return
	}

	//upload the backup to the cloud
	authStr := cloud.User + ":" + cloud.Password
	connStr = cloud.Address + cloud.User + "/" + cloud.Folder + "/" + filename
	out, err = exec.Command("curl", "-u", authStr, "-T", fpath, connStr).CombinedOutput()
	if err != nil {
		revel.AppLog.Error("failed to upload backup to cloud", "authStr", authStr, "fpath",
			fpath, "connStr", connStr, "err", err.Error(), "out", string(out))
		return
	}

	revel.AppLog.Warn("done creating DB backup...")
	return
}

/*fetchEnrollData moves the old file containing all courses of study into a backup folder,
fetches the newest version of that file and deletes all files in the backup folder that are
older than the last month. */
func fetchEnrollData() (err error) {

	//TODO: write this function less problem specific for a general file to be fetched
	//e.g. put the filename and the host in the config, set a custom backup folder, etc.

	//move old file in 'bak' folder
	now := time.Now().Format("2006-01-02_15:04:05")
	fname := "outdated_" + now + ".csv"
	source := filepath.Join(enrollFilePath, "enrollment-data-turm2.csv")
	dest := filepath.Join(enrollFilePath, "bak", fname)

	_, err = exec.Command("mv", source, dest).Output()
	if err != nil {
		revel.AppLog.Error("failed to move file into backup folder",
			"source", source, "dest", dest, "err", err.Error())
		return
	}

	//now get the new file
	host := "https://vdxo.rz.tu-ilmenau.de:8543/turm2/a504be42-6751-4627-ae37-54b81c863f76/enrollment-data-turm2.csv"
	_, err = exec.Command("wget", "-P", enrollFilePath, host).Output()
	if err != nil {
		revel.AppLog.Error("failed to fetch the new file", "path", enrollFilePath,
			"host", host, "err", err.Error())
		return
	}

	//get all files in the enrollment file backup folder
	cmd := exec.Command("ls", filepath.Join(enrollFilePath, "bak"))
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	//run command and check for error
	err = cmd.Run()
	if err != nil {
		revel.AppLog.Error("failed to list all files in the backup folder", "fpath",
			filepath.Join(enrollFilePath, "bak"), "errRun", err.Error(), "stdErr", stderr.String())
		return
	}

	lines := strings.Split(out.String(), "\n")

	//for each file in the backup folder
	for _, line := range lines {

		if line != "" { //ignore the . file

			linePart := strings.ReplaceAll(line, "outdated_", "")
			linePart = strings.ReplaceAll(linePart, ".csv", "")

			lastMonth := time.Now().Add(time.Duration(-(30 * 24)) * time.Hour).
				Format("2006-01-02_15:04:05")

			//delete the file, if older than approx. one month
			if linePart < lastMonth {
				_, err = exec.Command("rm", filepath.Join(enrollFilePath, "bak", line)).Output()
				if err != nil {
					revel.AppLog.Error("failed to delete file", "filepath",
						filepath.Join(enrollFilePath, "bak", line), "err", err.Error())
					return
				}
			}
		}
	}

	return
}

/*parseStudies parses the courses of studies of all ldap users. */
type parseStudies struct{}

/*Run executes the parse studies job. */
func (e parseStudies) Run() {

	if !revel.DevMode {
		if err := parse(); err != nil {
			SendErrorNote()
		}

	} else {
		revel.AppLog.Warn("not in production mode, skip parsing studies...")
	}
}

//parse the csv file containing all user studies and insert each line into the DB. */
func parse() (err error) {

	revel.AppLog.Warn("start parsing studies...")

	if err = fetchEnrollData(); err != nil {

		//TODO: get filename from config (see fetchEnrollData())
		fpath := filepath.Join(enrollFilePath, "enrollment-data-turm2.csv")

		//open the file containing all entries
		f, err := os.Open(fpath)
		if err != nil {
			revel.AppLog.Error("cannot open enrollment file", "filepath", fpath,
				"err", err.Error())
			return err
		}

		defer f.Close()
		//opened file, init csv reader
		r := csv.NewReader(f)
		r.Comma = ';'

		//start the transaction to insert each row of the csv file
		tx, err := Db.Beginx()
		if err != nil {
			revel.AppLog.Error("failed to begin tx", "error", err.Error())
			return err
		}

		//first untouch the studies table
		_, err = tx.Exec(stmtUntouchStudies)
		if err != nil {
			revel.AppLog.Error("failed to untouch studies table", "err", err.Error())
			tx.Rollback()
			return err
		}

		//read row and insert data
		for {

			record, err := r.Read()
			if err != nil {
				if err == io.EOF { //end of file

					//delete all untouched studies entries
					_, err = tx.Exec(stmtDeleteUntouched)
					if err != nil {
						revel.AppLog.Error("failed to delete untouched entries", "err", err.Error())
						tx.Rollback()
						return err
					}

					tx.Commit()
					err = nil
					revel.AppLog.Warn("finished parsing studies...")
					return err
				}

				revel.AppLog.Error("failed to read csv row", "err", err.Error())
				tx.Rollback()
				return err
			}

			//insert entry if row data is valid
			if len(record) > 3 {

				//insert degree
				_, err = tx.Exec(stmtInsertDegree, record[3])
				if err != nil {
					revel.AppLog.Error("failed to insert degree", "err", err.Error())
					tx.Rollback()
					return err
				}
			}

			//insert course of studies
			_, err = tx.Exec(stmtInsertCourseOfStudies, record[1])
			if err != nil {
				revel.AppLog.Error("failed to insert course of studies",
					"err", err.Error())
				tx.Rollback()
				return err
			}

			//convert the semester from string to int
			semester, err := strconv.Atoi(record[2])
			if err != nil {
				revel.AppLog.Error("semester type conversion error", "semester",
					record[2], "err", err.Error())
				tx.Rollback()
				return err
			}

			//convert the matr nr from string to int
			matrnr, err := strconv.Atoi(record[0])
			if err != nil {
				revel.AppLog.Error("matr nr type conversion error", "matr nr",
					record[0], "err", err.Error())
				tx.Rollback()
				return err
			}

			//insert the studies entry
			_, err = tx.Exec(stmtInsertStudies, matrnr, record[1], /*studies*/
				record[3] /*degree*/, semester)
			if err != nil {
				revel.AppLog.Error("failed to insert studies entry", "record", record,
					"matr nr", matrnr, "semester", semester, "err", err.Error())
				tx.Rollback()
				return err
			}
		} /* end of for */

	}

	return
}

//dbConnTest is a small job pinging the db in fixed intervals
type dbConnTest struct{}

/*Run the db connection test. */
func (e dbConnTest) Run() {

	revel.AppLog.Warn("running DB connection test...")
	err := Db.Ping()
	if err != nil {
		revel.AppLog.Error("DB connection test failed", "err", err.Error())
		SendErrorNote()
	}
}

//initJobData initializes all job config variables
func initJobData() {

	var found bool
	cloud.User = Mailer.User
	cloud.Password = Mailer.Password

	if cloud.Path, found = revel.Config.String("dbbackup.path"); !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "dbbackup.path")
	}
	if cloud.Address, found = revel.Config.String("dbbackup.ownCloud"); !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "dbbackup.ownCloud")
	}
	if cloud.Folder, found = revel.Config.String("dbbackup.ownCloudFolder"); !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "dbbackup.ownCloudFolder")
	}

	if enrollFilePath, found = revel.Config.String("enroll.data"); !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "enroll.data")
	}
}

//initJobSchedules initializes all execution times of jobs
func initJobSchedules() {

	jobSchedules = make(map[string]string)

	//DB backup
	backupDB, found := revel.Config.String("jobs.dbbackup")
	if !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "jobs.dbbackup")
	}
	jobSchedules["jobs.dbbackup"] = backupDB

	//parse studies
	studies, found := revel.Config.String("jobs.parseStudies")
	if !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "jobs.parseStudies")
	}
	jobSchedules["jobs.parseStudies"] = studies

	//ping DB
	connTest, found := revel.Config.String("jobs.connTest")
	if !found {
		revel.AppLog.Fatal("cannot find key in config", "key", "jobs.connTest")
	}
	jobSchedules["jobs.connTest"] = connTest
}

const (
	stmtUntouchStudies = `
    UPDATE studies
    SET touched = false
  `

	stmtDeleteUntouched = `
    DELETE FROM studies
    WHERE NOT touched
  `

	stmtInsertStudies = `
    INSERT INTO studies
      (user_id, course_of_studies_id, degree_id, semester, touched)

    (SELECT
      u.id AS user_id,

      (SELECT id
        FROM courses_of_studies
        WHERE name = $2)
      AS course_of_studies_id,

      (SELECT id FROM degrees WHERE name = $3)
      AS degree_id,

      $4 AS semester, true AS touched
    FROM users u
    WHERE u.matr_nr = $1)

    ON CONFLICT (user_id, course_of_studies_id, degree_id) DO UPDATE
    SET semester = $4, touched = true
  `

	stmtInsertDegree = `
    INSERT INTO degrees (name)
    VALUES ($1)
    ON CONFLICT (name) DO NOTHING
  `

	stmtInsertCourseOfStudies = `
    INSERT INTO courses_of_studies (name)
    VALUES ($1)
    ON CONFLICT (name) DO NOTHING
  `
)
