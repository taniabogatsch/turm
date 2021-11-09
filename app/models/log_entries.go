package models

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"os"
	"strings"
	"time"
	"turm/app"
)

/*LogEntries contains all relevant log entries. */
type LogEntries []LogEntry

/*LogEntry represents a line of the error log. */
type LogEntry struct {
	ID             int       `db:"id, primarykey, autoincrement"`
	TimeOfCreation time.Time `db:"time_of_creation"`
	JSON           string    `db:"json"`
	Solved         bool      `db:"solved"`

	TimeOfCreationStr string `db:"time_of_creation_str"`
}

/*Select all log entries. */
func (entries *LogEntries) Select() (err error) {

	err = app.Db.Select(entries, stmtSelectLogEntries, app.TimeZone)
	if err != nil {
		log.Error("failed to select log entries", "error", err.Error())
	}
	return
}

/*Insert opens the log file and inserts all new log entries. */
func (entries *LogEntries) Insert() (err error) {

	//open file in read-only mode
	file, err := os.Open(app.PathErrorLog)
	if err != nil {
		log.Error("failed to open error log file", "filepath", app.PathErrorLog,
			"error", err.Error())
		return
	}

	//new file scanner
	scanner := bufio.NewScanner(file)

	//config the scanner to read each line
	scanner.Split(bufio.ScanLines)

	//store each line in the jsons slice
	var jsons []string
	for scanner.Scan() {
		jsons = append(jsons, scanner.Text())
	}

	//close the file
	file.Close()

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return
	}

	//get the last extraction time
	var lastLogEntryTime sql.NullTime
	err = tx.Get(&lastLogEntryTime, stmtGetLastExtractionTime)
	if err != nil {
		log.Error("failed to get last extraction time", "error", err.Error())
		tx.Rollback()
		return
	}
	lastLogEntryTime.Time = lastLogEntryTime.Time.Add(time.Microsecond)

	//insert all log entries occuring after the last extraction time
	for _, line := range jsons {

		//unmarshal line
		var jsonLine map[string]interface{}
		err = json.Unmarshal([]byte(line), &jsonLine)
		if err != nil {
			log.Error("failed to unmarshal line", "line", line, "error", err.Error())
			tx.Rollback()
			return
		}

		if jsonLine["t"] == nil {
			log.Error("failed to get time of creation of log entry", "line", line)
			tx.Rollback()
			return
		}

		if jsonLine["caller"] != nil {
			caller := jsonLine["caller"].(string)
			if strings.Contains(caller, "revel_logger.go:39") || strings.Contains(caller,
				"compress.go:151") || strings.Contains(caller, "results.go:428") {
				continue
			}
		}

		//format: 2021-03-09T09:10:48.033279498+01:00
		timeOfCreation, err := time.Parse("2006-01-02T15:04:05.999999999-07:00", jsonLine["t"].(string))
		if err != nil {
			log.Error("failed to parse time of creation", "t", jsonLine["t"].(string),
				"error", err.Error())
			tx.Rollback()
			return err
		}

		if lastLogEntryTime.Time.Before(timeOfCreation) &&
			!lastLogEntryTime.Time.Equal(timeOfCreation) {

			_, err = tx.Exec(stmtInsertLogEntry, timeOfCreation, line)
			if err != nil {
				log.Error("failed to insert log entry", "timeOfCreation", timeOfCreation,
					"line", line, "error", err.Error())
				tx.Rollback()
				return err
			}
		}
	}

	tx.Commit()
	return
}

/*Solve a log entry. */
func (entry *LogEntry) Solve() (err error) {

	_, err = app.Db.Exec(stmtSolveLogEntry, entry.ID)
	if err != nil {
		log.Error("failed to solve log entry", "entryID", entry.ID,
			"error", err.Error())
	}

	return
}

const (
	stmtSelectLogEntries = `
    SELECT id, time_of_creation, json,
			TO_CHAR (time_of_creation AT TIME ZONE $1, 'YYYY-MM-DD HH24:MI:SS') AS time_of_creation_str
    FROM log_entries
		WHERE NOT solved
    ORDER BY time_of_creation DESC
  `

	stmtSolveLogEntry = `
    UPDATE log_entries
    SET solved = true
    WHERE id = $1
  `

	stmtGetLastExtractionTime = `
		SELECT MAX(time_of_creation) AS last_log_entry_time
		FROM log_entries
	`

	stmtInsertLogEntry = `
		INSERT INTO log_entries
			(time_of_creation, json)
		VALUES ($1, $2)
	`
)
