package models

import (
	"database/sql"

	"github.com/revel/revel"
)

/*Category is a model of the category table. Categories are
used for FAQs (faq_category) and the news feed (news_feed_category). */
type Category struct {
	ID           int           `db:"id, primarykey, autoincrement"`
	Name         string        `db:"name"`
	Creator      sql.NullInt32 `db:"creator"`
	CreationDate string        `db:"creationdate"`
}

/*Validate Category fields. */
func (category *Category) Validate(v *revel.Validation) {
	//TODO
}

/*FAQ is a model of the faq table. */
type FAQ struct {
	ID           int           `db:"id, primarykey, autoincrement"`
	Creator      sql.NullInt32 `db:"creator"`
	CategoryID   int           `db:"categoryid"`
	Question     string        `db:"question"`
	Answer       string        `db:"answer"`
	CreationDate string        `db:"creationdate"`
}

/*Validate FAQ fields. */
func (faq *FAQ) Validate(v *revel.Validation) {
	//TODO
}

/*NewsFeed is a model of the news_feed table. */
type NewsFeed struct {
	ID           int           `db:"id, primarykey, autoincrement"`
	Creator      sql.NullInt32 `db:"creator"`
	CategoryID   int           `db:"categoryid"`
	Content      string        `db:"content"`
	CreationDate string        `db:"creationdate"`
}

/*Validate NewsFeed fields. */
func (newsFeed *NewsFeed) Validate(v *revel.Validation) {
	//TODO
}
