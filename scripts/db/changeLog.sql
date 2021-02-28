
/* Add enrollment comments. */
ALTER TABLE enrolled ADD COLUMN comment varchar(511);
ALTER TABLE events ADD COLUMN has_comments boolean NOT NULL DEFAULT false;
