
/* Add enrollment comments. */
ALTER TABLE enrolled ADD COLUMN comment varchar(511);
ALTER TABLE events ADD COLUMN has_comments boolean NOT NULL DEFAULT false;

/* View log entries at admin page. */
CREATE TABLE log_entries (
  ID                  serial                        PRIMARY KEY,
  time_of_creation    timestamp with time zone      NOT NULL,
  json                text                          NOT NULL,
  solved              boolean                       NOT NULL DEFAULT false
);
COMMENT ON TABLE log_entries IS 'Table containing all relevant error log entries.';
