package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/revel/revel"
)

var (
	//cloud holds all DB backup cloud data
	cloud cloudConn

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
	revel.AppLog.Warn("running job to backup DB...")
	backup()
}

//backup the DB locally and upload it to the specified upload location
func backup() (success bool) {

	revel.AppLog.Warn("start creating DB backup...")

	//connection
	connStr := "--dbname=postgresql://" + DBData.User + ":" + DBData.Password +
		"@" + DBData.Host + ":" + DBData.Port + "/" + DBData.Name

	//file config
	now := time.Now().Format("2006-01-02_15:04:05")
	filename := now + "_DBdump.sql"
	fpath := filepath.Join(cloud.Path, filename)
	revel.AppLog.Warn("Filepath: " + fpath)

	//create a local backup
	out, err := exec.Command("pg_dump", "--no-owner", connStr, "-f", fpath).CombinedOutput()
	if err != nil {
		revel.AppLog.Error(err.Error())
		revel.AppLog.Error(string(out))
		return
	}

	//upload the backup to the cloud
	authStr := cloud.User + ":" + cloud.Password
	connStr = cloud.Address + cloud.User + "/" + cloud.Folder + "/" + filename
	out, err = exec.Command("curl", "-u", authStr, "-T", fpath, connStr).CombinedOutput()
	if err != nil {
		revel.AppLog.Error(err.Error())
		revel.AppLog.Error(string(out))
		return
	}

	revel.AppLog.Warn("done creating DB backup...")
	return true
}

//initCloudData initializes all cloud config variables
func initCloudData() {

	var found bool
	cloud.User = Mailer.User
	cloud.Password = Mailer.Password

	if cloud.Path, found = revel.Config.String("dbbackup.path"); !found {
		revel.AppLog.Error("no dbbackup.path set in config")
		os.Exit(1)
	}
	if cloud.Address, found = revel.Config.String("dbbackup.ownCloud"); !found {
		revel.AppLog.Error("no dbbackup.ownCloud set in config")
		os.Exit(1)
	}
	if cloud.Folder, found = revel.Config.String("dbbackup.ownCloudFolder"); !found {
		revel.AppLog.Error("no dbbackup.ownCloudFolder set in config")
		os.Exit(1)
	}
}

//initJobSchedules initializes all execution times of jobs
func initJobSchedules() {

	jobSchedules = make(map[string]string)

	backupDB, found := revel.Config.String("jobs.dbbackup")
	if !found {
		revel.AppLog.Error("no jobs.dbbackup set in config")
		os.Exit(1)
	}
	jobSchedules["jobs.dbbackup"] = backupDB
}
