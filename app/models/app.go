package models

import (
	"database/sql"

	"github.com/revel/revel"
)

/*Category contains all directly category related values. Categories are
used for FAQs (faq_category) and the news feed (news_feed_category). */
type Category struct {
	ID           int           `db:"id, primarykey, autoincrement"`
	Name         string        `db:"name"`
	Creator      sql.NullInt32 `db:"creator"`
	CreationDate string        `db:"creationdate"`
}

/*Validate validates the Category struct fields. */
func (category *Category) Validate(v *revel.Validation) {
	//TODO
}

/*FAQ contains all directly FAQ related values. */
type FAQ struct {
	ID           int           `db:"id, primarykey, autoincrement"`
	Creator      sql.NullInt32 `db:"creator"`
	CategoryID   int           `db:"categoryid"`
	Question     string        `db:"question"`
	Answer       string        `db:"answer"`
	CreationDate string        `db:"creationdate"`
}

/*Validate validates the FAQ struct fields. */
func (faq *FAQ) Validate(v *revel.Validation) {
	//TODO
}

/*NewsFeed contains all directly news feed related values. */
type NewsFeed struct {
	ID           int           `db:"id, primarykey, autoincrement"`
	Creator      sql.NullInt32 `db:"creator"`
	CategoryID   int           `db:"categoryid"`
	Content      string        `db:"content"`
	CreationDate string        `db:"creationdate"`
}

/*Validate validates the NewsFeed struct fields. */
func (newsFeed *NewsFeed) Validate(v *revel.Validation) {
	//TODO
}
