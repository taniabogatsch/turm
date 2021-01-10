# Welcome to Turm

Turm2 is an enrollment system allowing users to enroll in courses. There is no official release yet. It is currently running at [Turm2](https://turm2.tu-ilmenau.de).

It uses:
- [Go](https://github.com/golang/go)
- [Revel](https://github.com/revel/)
- [PostgreSQL](https://www.postgresql.org/)
- [Postfix](http://www.postfix.org/)
- [Bootstrap 4.4.1](https://getbootstrap.com)
- [Bootstrap Icons](https://icons.getbootstrap.com)
- [JQuery 3.4.1](https://jquery.com)
- [Quill](https://quilljs.com) 
- [jmoiron/sqlx](https://github.com/jmoiron/sqlx)
- [k3a/html2text](https://github.com/k3a/html2text)
- [ldap.v2](https://gopkg.in/ldap.v2)

## Usage

### Set up a PostgreSQL database

```
create database turm;
create user turm with encrypted password 'your_password';
grant all privileges on database turm to turm;
```
With the postgreSQL superuser, add the `pgcrypto` extension (`create extension pgcrypto`).

Create the DB schema using `scripts/db/create.sql`.

```
psql -h localhost -d turm -U turm -p 5432 -f create.sql
```

### Set up Postfix

Install [Postfix](http://www.postfix.org/). Choose `internet with smarthost` and set all required parameters.

```
sudo apt install postfix
```

For validation, you can do the following, where `your-mail@abc.de` is the receiving e-mail address.

```
sudo apt install mailutils
mailx -s'foobar' your-mail@abc.de -r'abc@yourhost.de'
```

### Start the web server (development)

Install [Go](https://github.com/golang/go) and [Git](https://git-scm.com/).

Use [Git](https://git-scm.com/) to clone the repository. 

```
cd ~/go/src
git clone https://github.com/taniabogatsch/turm.git 
or (depending on your repo access rights)
git clone git@github.com:taniabogatsch/turm.git
```

Now use `go get` to install [Revel](https://github.com/revel/).

```
go get -u github.com/revel/cmd/revel
```

In your `~/.profile` file, add the following line:

```
export PATH="$PATH:~/go/bin"
```

If you want to, you can verify your revel installation.

```
source $HOME/.profile
revel new -a my-app -r
```

Install the following missing packages.

```
go get -u github.com/jmoiron/sqlx
go get -u github.com/jackc/pgx/stdlib
go get -u gopkg.in/ldap.v2
go get -u github.com/k3a/html2text
```

Create the following folders and files:

```
mkdir ~/go/src/db_backups
mkdir ~/go/src/studiengaenge
vim ~/go/src/studiengaenge/enrollment-data-turm2.csv
```

Adjust all config values `app/conf/app.conf`. See below for a detailed description.

Create a `passwords.json` file at `app/conf/`. It should only contain the following two values:
```
{
  "db.pw": "your_password",
  "email.pw": "your_password"
}
```

### Run or deploy

Run with `revel run turm` or create a `run.sh` with `revel package turm prod`.

## Code Layout

The directory structure of a generated Revel application:

    app/                   App sources
         auth              Authentication against the LDAP server
         controllers/      GET, POST, etc. controllers
         models/           DB models
         views/            HTML templates and some JS
         conf.go           Init all conf values
         db.go             Init all DB values
         init.go           App initialization, e.g. opens a DB connection
         jobs.go           Run scheduled jobs
         mailer.go         Sends e-mails

    conf/                  Configuration directory
         app.conf          Main app configuration file
         passwords.json    Map of passwords, not included in the repository
         routes            Routes definition file

    messages/              Message files, currently supported languages are en-US and de-DE
    modules/jobs/          Chron jobs

    public/                Public static assets
         css/              CSS files
         js/               Javascript files
         images/           Image files
    
    scripts/db/            DB schema

    tests/                 Test suites
    
# Effective Go

https://golang.org/doc/effective_go.html

* Every package should have a **package comment**, a block comment preceding the package clause. For multi-file packages, the package comment only needs to be present in one file, and any one will do.
* Inside a package, any comment immediately preceding a top-level declaration serves as a **doc comment** for that declaration. Every exported (capitalized) name in a program should have a doc comment. The first sentence should be a one-sentence summary that starts with the name being declared. **Addition**: doc comments should be surrounded by `/* ... */`.
* By convention, **packages are given lower case, single-word names**. It's helpful if everyone using the package can use the same name to refer to its contents, which implies that the package name should be good: short, concise, evocative.
* Finally, the convention in Go is to use **MixedCaps or mixedCaps** rather than underscores to write multiword names.
