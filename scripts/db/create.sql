/* Database schema, 2020-04-14 */


CREATE TABLE users (
  id              serial                    PRIMARY KEY,
  last_name       varchar(255)              NOT NULL,
  first_name      varchar(255)              NOT NULL,
  email           varchar(255)              UNIQUE NOT NULL,
  salutation      integer                   NOT NULL,
  role            integer                   NOT NULL,
  last_login      timestamp with time zone  NOT NULL,
  first_login     timestamp with time zone  NOT NULL,
  language        varchar(63),
  matr_nr         integer                   UNIQUE,
  affiliations    varchar(127)[],
  academic_title  varchar(127),
  title           varchar(127),
  name_affix      varchar(127),
  password        varchar(511),
  activation_code varchar(255)
);


CREATE TABLE degrees (
  id    serial        PRIMARY KEY,
  name  varchar(255)  NOT NULL UNIQUE
);


CREATE TABLE courses_of_studies (
  id    serial        PRIMARY KEY,
  name  varchar(511)  NOT NULL UNIQUE
);


CREATE TABLE studies (
  user_id                 integer   NOT NULL,
  semester                integer   NOT NULL,
  degree_id               integer   NOT NULL,
  course_of_studies_id    integer   NOT NULL,
  touched                 bool      NOT NULL,

  PRIMARY KEY (user_id, degree_id, course_of_studies_id),
  FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
  FOREIGN KEY (degree_id) REFERENCES degrees (id) ON DELETE CASCADE,
  FOREIGN KEY (course_of_studies_id) REFERENCES courses_of_studies (id) ON DELETE CASCADE
);


CREATE TABLE groups (
  id              serial                    PRIMARY KEY,
  parent_id       integer,
  name            varchar(255)              NOT NULL,
  course_limit    integer,
  last_editor     integer, /* Set to null if user data is deleted due to data policy requirements. */
  last_edited     timestamp with time zone  NOT NULL,

  FOREIGN KEY (last_editor) REFERENCES users (id) ON DELETE SET NULL,
  FOREIGN KEY (parent_id) REFERENCES groups (id) /* Prevent the deletion of groups if they still have subgroups. */
);


CREATE TABLE courses (
  id                    serial                    PRIMARY KEY,
  title                 varchar(511)              NOT NULL,
  creator               integer, /* Set to null if user data is deleted due to data policy requirements. */
  subtitle              varchar(511),
  visible               boolean                   NOT NULL,
  active                boolean                   NOT NULL,
  only_ldap             boolean                   NOT NULL,
  creation_date         timestamp with time zone  NOT NULL,
  description           text,
  speaker               text,
  fee                   numeric,
  custom_email          text,
  enroll_limit_events   integer,
  enrollment_start      timestamp with time zone  NOT NULL,
  enrollment_end        timestamp with time zone  NOT NULL,
  unsubscribe_end       timestamp with time zone,
  expiration_date       timestamp with time zone  NOT NULL,
  parent_id             integer,

  FOREIGN KEY (creator) REFERENCES users (id) ON DELETE SET NULL,
  FOREIGN KEY (parent_id) REFERENCES groups (id) /* Prevent the deletion of groups if they still have courses. */
);


CREATE TABLE events (
  id                serial        PRIMARY KEY,
  course_id         integer       NOT NULL,
  capacity          integer       NOT NULL,
  has_waitlist      boolean       NOT NULL,
  title             varchar(255)  NOT NULL,
  annotation        varchar(255),
  enrollment_key    varchar(511),

  FOREIGN KEY (course_id) REFERENCES courses (id) ON DELETE CASCADE
);


CREATE TABLE meetings (
  id                serial                    PRIMARY KEY,
  event_id          integer                   NOT NULL,
  meeting_interval  integer                   NOT NULL,
  weekday           integer,
  place             varchar(255),
  annotation        varchar(255),
  meeting_start     timestamp with time zone  NOT NULL,
  meeting_end       timestamp with time zone  NOT NULL,

  FOREIGN KEY (event_id) REFERENCES events (id) ON DELETE CASCADE
);


CREATE TABLE editors (
  user_id         integer   NOT NULL,
  course_id       integer   NOT NULL,
  view_matr_nr    boolean   NOT NULL,

  PRIMARY KEY (user_id, course_id),
  FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
  FOREIGN KEY (course_id) REFERENCES courses (id) ON DELETE CASCADE
);


CREATE TABLE instructors (
  user_id         integer   NOT NULL,
  course_id       integer   NOT NULL,
  view_matr_nr    boolean   NOT NULL,

  PRIMARY KEY (user_id, course_id),
  FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
  FOREIGN KEY (course_id) REFERENCES courses (id) ON DELETE CASCADE
);


CREATE TABLE blacklists (
  user_id    integer   NOT NULL,
  course_id  integer   NOT NULL,

  PRIMARY KEY (user_id, course_id),
  FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
  FOREIGN KEY (course_id) REFERENCES courses (id) ON DELETE CASCADE
);


CREATE TABLE whitelists (
  user_id    integer   NOT NULL,
  course_id  integer   NOT NULL,

  PRIMARY KEY (user_id, course_id),
  FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
  FOREIGN KEY (course_id) REFERENCES courses (id) ON DELETE CASCADE
);


CREATE TABLE enrollment_restrictions (
  id                        serial    PRIMARY KEY,
  course_id                 integer   NOT NULL,
  minimum_semester          integer,
  degree_id                 integer,
  courses_of_studies_id     integer,

  FOREIGN KEY (course_id) REFERENCES courses (id) ON DELETE CASCADE,
  FOREIGN KEY (degree_id) REFERENCES degrees (id) ON DELETE CASCADE,
  FOREIGN KEY (courses_of_studies_id) REFERENCES courses_of_studies (id) ON DELETE CASCADE
);


CREATE TABLE enrolled (
  user_id               integer                   NOT NULL,
  event_id              integer                   NOT NULL,
  status                integer                   NOT NULL,
  email_traffic         boolean                   NOT NULL false,
  time_of_enrollment    timestamp with time zone  NOT NULL DEFAULT now(),

  PRIMARY KEY(user_id, event_id),
  FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
  FOREIGN KEY (event_id) REFERENCES events (id) ON DELETE CASCADE
);


CREATE TABLE unsubscribed (
  user_id    integer   NOT NULL,
  event_id   integer   NOT NULL,

  PRIMARY KEY (user_id, event_id),
  FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
  FOREIGN KEY (event_id) REFERENCES events (id) ON DELETE CASCADE
);


CREATE TABLE faq_category (
  id              serial                    PRIMARY KEY,
  name            varchar(255)              NOT NULL,
  last_editor     integer, /* Set to null if user data is deleted due to data policy requirements. */
  last_edited     timestamp with time zone  NOT NULL,

  FOREIGN KEY (last_editor) REFERENCES users (id) ON DELETE SET NULL
);


CREATE TABLE faqs (
  id              serial                    PRIMARY KEY,
  last_editor     integer, /* Set to null if user data is deleted due to data policy requirements. */
  category_id     integer                   NOT NULL,
  question        varchar(511)              NOT NULL,
  answer          text                      NOT NULL,
  last_edited     timestamp with time zone  NOT NULL,

  FOREIGN KEY (last_editor) REFERENCES users (id) ON DELETE SET NULL,
  FOREIGN KEY (category_id) REFERENCES faq_category (id) /* Prevent the deletion of categories if they still contain FAQs. */
);


CREATE TABLE news_feed_category (
  id              serial                    PRIMARY KEY,
  name            varchar(255)              NOT NULL,
  last_editor     integer, /* Set to null if user data is deleted due to data policy requirements. */
  last_edited     timestamp with time zone  NOT NULL,

  FOREIGN KEY (last_editor) REFERENCES users (id) ON DELETE SET NULL
);


CREATE TABLE news_feed (
  id              serial                    PRIMARY KEY,
  last_editor     integer, /* Set to null if user data is deleted due to data policy requirements. */
  category_id     integer                   NOT NULL,
  content         text                      NOT NULL,
  last_edited     timestamp with time zone  NOT NULL,

  FOREIGN KEY (last_editor) REFERENCES users (id) ON DELETE SET NULL,
  FOREIGN KEY (category_id) REFERENCES news_feed_category (id) /* Prevent the deletion of categories if they still contain news. */
);


CREATE TABLE calendar_events (
  id              serial                    PRIMARY KEY,
  course_id       integer                   NOT NULL,
  title           varchar(255)              NOT NULL,
  annotation      varchar(255),

  FOREIGN KEY (course_id) REFERENCES courses (id) ON DELETE CASCADE
);


CREATE TABLE day_templates (
  id                  serial                      PRIMARY KEY,
  calendar_event_id   integer                     NOT NULL,
  start_time          time with time zone         NOT NULL,
  end_time            time with time zone         NOT NULL,
  intervall           integer                     NOT NULL DEFAULT 60,
  day_of_week         integer                     NOT NULL,
  active              boolean                     DEFAULT true,
  deactivation_date   timestamp with time zone,

  FOREIGN KEY (calendar_event_id) REFERENCES calendar_events (id) ON DELETE CASCADE
);


CREATE TABLE slots (
  id                  serial                    PRIMARY KEY,
  user_id             integer                   NOT NULL,
  day_tmpl_id         integer                   NOT NULL,
  start_time          timestamp with time zone  NOT NULL,
  end_time            timestamp with time zone  NOT NULL,
  created             timestamp with time zone  DEFAULT now(),

  FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
  FOREIGN KEY (day_tmpl_id) REFERENCES day_templates (id) ON DELETE CASCADE
);


CREATE TABLE calendar_exceptions (
  id                  serial                    PRIMARY KEY,
  calendar_event_id   integer                     NOT NULL,
  start_time          timestamp with time zone,
  end_time            timestamp with time zone,
  annotations         varchar(255),

  FOREIGN KEY (calendar_event_id) REFERENCES calendar_events (id) ON DELETE CASCADE
);
