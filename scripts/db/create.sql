/* Database schema, 2020-04-14 */


CREATE TYPE salutation AS ENUM (
    'mr',
    'ms',
    'none'
);
CREATE TYPE role AS ENUM (
    'user',
    'creator',
    'admin'
);
CREATE TYPE meetinginterval AS ENUM (
    'single',
    'weekly',
    'even',
    'odd'
);
CREATE TYPE status AS ENUM (
    'enrolled',
    'on waitlist',
    'awaiting payment',
    'paid',
    'freed'
);


CREATE TABLE users (
  id              serial                    PRIMARY KEY,
  lastname        varchar(255)              NOT NULL,
  firstname       varchar(255)              NOT NULL,
  email           varchar(255)              UNIQUE NOT NULL,
  salutation      salutation                NOT NULL,
  role            role                      NOT NULL,
  lastlogin       timestamp with time zone  NOT NULL,
  firstlogin      timestamp with time zone  NOT NULL,
  matrnr          integer                   UNIQUE,
  affiliation     varchar(255)[],
  academictitle   varchar(127),
  title           varchar(127),
  nameaffix       varchar(127),
  pw              varchar(511),
  activationcode  varchar(255)
);


CREATE TABLE degree (
  id    serial        PRIMARY KEY,
  name  varchar(255)  NOT NULL UNIQUE
);
CREATE TABLE courseofstudies (
  id    serial        PRIMARY KEY,
  name  varchar(511)  NOT NULL UNIQUE
);


CREATE TABLE studies (
  userid              integer   NOT NULL,
  semester            integer   NOT NULL,
  degreeid            integer   NOT NULL,
  courseofstudiesid   integer   NOT NULL,

  PRIMARY KEY (userid, degreeid, courseofstudiesid),
  FOREIGN KEY (userid) REFERENCES users (id) ON DELETE CASCADE,
  FOREIGN KEY (degreeid) REFERENCES degree (id) ON DELETE CASCADE,
  FOREIGN KEY (courseofstudiesid) REFERENCES courseofstudies (id) ON DELETE CASCADE
);


CREATE TABLE course (
  id                  serial                    PRIMARY KEY,
  title               varchar(511)              NOT NULL,
  creator             integer, /* Set to null if user data is deleted due to data policy requirements. */
  subtitle            varchar(511),
  visible             boolean                   NOT NULL,
  active              boolean                   NOT NULL,
  onlyldap            boolean                   NOT NULL,
  creationdate        timestamp with time zone  NOT NULL,
  description         text,
  fee                 real,
  customemail         text,
  enrolllimitevents   integer,
  enrollmentstart     timestamp with time zone  NOT NULL,
  enrollmentend       timestamp with time zone  NOT NULL,
  unsubscribeend      timestamp with time zone,
  expirationdate      timestamp with time zone  NOT NULL,

  FOREIGN KEY (creator) REFERENCES users (id)
);


CREATE TABLE event (
  id                serial        PRIMARY KEY,
  courseid          integer       NOT NULL,
  capacity          integer       NOT NULL,
  haswaitlist       boolean       NOT NULL,
  title             varchar(255)  NOT NULL,
  description       text,
  registrationkey   varchar(512),

  FOREIGN KEY (courseid) REFERENCES course (id) ON DELETE CASCADE
);


CREATE TABLE meeting (
  id              serial                    PRIMARY KEY,
  eventid         integer                   NOT NULL,
  meetinginterval meetinginterval           NOT NULL,
  weekday         integer,
  place           varchar(255),
  annotation      varchar(255),
  meetingstart    timestamp with time zone  NOT NULL,
  meetingend      timestamp with time zone  NOT NULL,

  FOREIGN KEY (eventid) REFERENCES event (id) ON DELETE CASCADE
);


CREATE TABLE editor (
  userid    integer   NOT NULL,
  courseid  integer   NOT NULL,

  PRIMARY KEY (userid, courseid),
  FOREIGN KEY (userid) REFERENCES users (id) ON DELETE CASCADE,
  FOREIGN KEY (courseid) REFERENCES course (id) ON DELETE CASCADE
);


CREATE TABLE instructor (
  userid    integer   NOT NULL,
  courseid  integer   NOT NULL,

  PRIMARY KEY (userid, courseid),
  FOREIGN KEY (userid) REFERENCES users (id) ON DELETE CASCADE,
  FOREIGN KEY (courseid) REFERENCES course (id) ON DELETE CASCADE
);


CREATE TABLE blacklist (
  userid    integer   NOT NULL,
  courseid  integer   NOT NULL,

  PRIMARY KEY (userid, courseid),
  FOREIGN KEY (userid) REFERENCES users (id) ON DELETE CASCADE,
  FOREIGN KEY (courseid) REFERENCES course (id) ON DELETE CASCADE
);


CREATE TABLE whitelist (
  userid    integer   NOT NULL,
  courseid  integer   NOT NULL,

  PRIMARY KEY (userid, courseid),
  FOREIGN KEY (userid) REFERENCES users (id) ON DELETE CASCADE,
  FOREIGN KEY (courseid) REFERENCES course (id) ON DELETE CASCADE
);


CREATE TABLE enrollment_restrictions (
  id                  serial    PRIMARY KEY,
  courseid            integer   NOT NULL,
  minimumsemester     integer   NOT NULL,
  degreeid            integer   NOT NULL,
  courseofstudiesid   integer   NOT NULL,

  FOREIGN KEY (courseid) REFERENCES course (id) ON DELETE CASCADE,
  FOREIGN KEY (degreeid) REFERENCES degree (id) ON DELETE CASCADE,
  FOREIGN KEY (courseofstudiesid) REFERENCES courseofstudies (id) ON DELETE CASCADE
);


CREATE TABLE enrolled (
  userid            integer                   NOT NULL,
  eventid           integer                   NOT NULL,
  status            status                    NOT NULL,
  emailtraffic      boolean                   NOT NULL,
  timeofenrollment  timestamp with time zone  NOT NULL,

  PRIMARY KEY(userid, eventid),
  FOREIGN KEY (userid) REFERENCES users (id) ON DELETE CASCADE,
  FOREIGN KEY (eventid) REFERENCES event (id) ON DELETE CASCADE
);


CREATE TABLE unsubscribed (
  userid    integer   NOT NULL,
  eventid   integer   NOT NULL,

  PRIMARY KEY (userid, eventid),
  FOREIGN KEY (userid) REFERENCES users (id) ON DELETE CASCADE,
  FOREIGN KEY (eventid) REFERENCES event (id) ON DELETE CASCADE
);


CREATE TABLE groups (
  id            serial                    PRIMARY KEY,
  parentid      integer,
  courseid      integer,
  name          varchar(255)              NOT NULL,
  maxcourses    integer,
  creator       integer, /* Set to null if user data is deleted due to data policy requirements. */
  creationdate  timestamp with time zone  NOT NULL,

  FOREIGN KEY (creator) REFERENCES users (id),
  FOREIGN KEY (courseid) REFERENCES course (id)
);


CREATE TABLE faq_category (
  id            serial                    PRIMARY KEY,
  name          varchar(255)              NOT NULL,
  creator       integer, /* Set to null if user data is deleted due to data policy requirements. */
  creationdate  timestamp with time zone  NOT NULL,

  FOREIGN KEY (creator) REFERENCES users (id)
);


CREATE TABLE faq (
  id            serial                    PRIMARY KEY,
  creator       integer, /* Set to null if user data is deleted due to data policy requirements. */
  categoryid    integer                   NOT NULL,
  question      varchar(511)              NOT NULL,
  answer        text                      NOT NULL,
  creationdate  timestamp with time zone  NOT NULL,

  FOREIGN KEY (creator) REFERENCES users (id),
  FOREIGN KEY (categoryid) REFERENCES faq_category (id)
);


CREATE TABLE news_feed_category (
  id            serial                    PRIMARY KEY,
  name          varchar(255)              NOT NULL,
  creator       integer, /* Set to null if user data is deleted due to data policy requirements. */
  creationdate  timestamp with time zone  NOT NULL,

  FOREIGN KEY (creator) REFERENCES users (id)
);


CREATE TABLE news_feed (
  id            serial                    PRIMARY KEY,
  creator       integer, /* Set to null if user data is deleted due to data policy requirements. */
  categoryid    integer                   NOT NULL,
  content       text                      NOT NULL,
  creationdate  timestamp with time zone  NOT NULL,

  FOREIGN KEY (creator) REFERENCES users (id),
  FOREIGN KEY (categoryid) REFERENCES news_feed_category (id)
);
