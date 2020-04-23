# Welcome to Turm2

Turm2 is an enrollment system allowing users to enroll in courses. There is no official release yet. An older version is currently deployed at [Turm2](https://turm2.tu-ilmenau.de). This project provides a new design as well as extended functionality.

It uses [Go](https://github.com/golang/go) and [Revel](https://github.com/revel/).

## Usage

### Set up a PostgreSQL database

```
create database turm;
create user turm with encrypted password 'turmpw';
grant all privileges on database turm to turm;
```
Import the database ERD, which is `erd.sql`.

Execute the DB Change Log, which is `db_changes.sql`.

### Start the web server:

```
go get -u github.com/revel/cmd/revel

go get -u github.com/jmoiron/sqlx
go get -u gopkg.in/ldap.v2
```

### Run or deploy the application: 

```
revel run turm
```
## Code Layout

The directory structure of a generated Revel application:

    app/                   App sources
         auth              Authentication against the LDAP server
         controllers/      GET, POST, etc. controllers
         models/           DB models
         views/            HTML templates and some JS
         init.go           App initialization, e.g. opens a DB connection
         mailer.go         Sends e-mails

    conf/                  Configuration directory
         app.conf          Main app configuration file
         passwords.json    Map of passwords, not included in the repository
         routes            Routes definition file

    messages/              Message files, currently supported languages are en-US and de-DE
    modules/jobs/          Chron jobs

    public/                Public static assets
        css/               CSS files
        js/                Javascript files
        images/            Image files
    
    scripts/db/create.sql  DB schema

    tests/                 Test suites
