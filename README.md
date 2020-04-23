# Welcome to Turm2

Turm2 is an enrollment system allowing users to enroll in courses. There is no official release yet. An older version is currently deployed at [Turm2](https://turm2.tu-ilmenau.de). This project provides a new design as well as extended functionality.

It uses [Go](https://github.com/golang/go), [Revel](https://github.com/revel/), [Bootstrap 4.4.1](https://getbootstrap.com), Bootstrap Icons, [JQuery 3.4.1](https://jquery.com) and the [CKEditor](https://ckeditor.com) in addition to the go packages mentioned below.

## Usage

### Set up a PostgreSQL database

```
create database turm;
create user turm with encrypted password 'turmpw';
grant all privileges on database turm to turm;
```
With the postgreSQL superuser, add the `pgcrypto` extension.

Create the DB schema using `scripts/db/create.sql`.

### Start the web server

Requires [Go](https://github.com/golang/go).

```
cd $GOPATH
go get -u github.com/revel/cmd/revel

go get -u github.com/jmoiron/sqlx
go get -u github.com/jackc/pgx/stdlib
go get -u gopkg.in/ldap.v2
go get -u github.com/k3a/html2text

cd src
git clone https://github.com/taniabogatsch/turm.git
```

Adjust all config values `app/conf/app.conf`.

### Run or deploy

Run with `revel run turm` or create a `run.sh` with `revel package turm prod`.

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
    
# Effective Go

https://golang.org/doc/effective_go.html

* Every package should have a **package comment**, a block comment preceding the package clause. For multi-file packages, the package comment only needs to be present in one file, and any one will do.
* Inside a package, any comment immediately preceding a top-level declaration serves as a **doc comment** for that declaration. Every exported (capitalized) name in a program should have a doc comment. The first sentence should be a one-sentence summary that starts with the name being declared. **Addition**: doc comments should be surrounded by `/* ... */`.
* By convention, **packages are given lower case, single-word names**. It's helpful if everyone using the package can use the same name to refer to its contents, which implies that the package name should be good: short, concise, evocative.
* Finally, the convention in Go is to use **MixedCaps or mixedCaps** rather than underscores to write multiword names.
