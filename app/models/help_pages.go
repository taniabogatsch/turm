package models

import (
	"database/sql"
	"strconv"
	"strings"
	"time"
	"turm/app"

	"github.com/revel/revel"
)

/*Category is a model of the category table. Categories are
used for grouping the entries of the FAQs (faq_category) and
the news feed (news_feed_category). */
type Category struct {
	ID         int           `db:"id, primarykey, autoincrement"`
	Name       string        `db:"name"`
	LastEditor sql.NullInt32 `db:"last_editor"`
	LastEdited string        `db:"last_edited"`

	Entries HelpPageEntries ``
}

/*Validate Category fields. */
func (category *Category) Validate(v *revel.Validation) {

	category.Name = strings.TrimSpace(category.Name)
	v.Check(category.Name,
		revel.MinSize{3},
		revel.MaxSize{255},
	).MessageKey("validation.invalid.text.short")
}

/*Insert a new category into either faq_category or news_feed_category. */
func (category *Category) Insert(table, userIDSession *string) (err error) {

	stmt := `
		INSERT INTO ` + *table + `
			(name, last_editor, last_edited)
		VALUES ($1, $2, $3)
		RETURNING id, name
	`

	userID, err := strconv.Atoi(*userIDSession)
	if err != nil {
		log.Error("failed to parse userID from userIDSession",
			"userIDSession", *userIDSession, "error", err.Error())
		return
	}

	err = app.Db.Get(category, stmt, category.Name, userID,
		time.Now().Format(revel.TimeFormats[0]))
	if err != nil {
		log.Error("failed to add category", "category", category, "userID", userID,
			"table", *table, "error", err.Error())
	}
	return
}

/*Update a category in either faq_category or news_feed_category. */
func (category *Category) Update(table, userIDSession *string) (err error) {

	stmt := `
		UPDATE ` + *table + `
		SET name = $1, last_editor = $2, last_edited = $3
		WHERE id = $4
		RETURNING id, name
	`

	userID, err := strconv.Atoi(*userIDSession)
	if err != nil {
		log.Error("failed to parse userID from userIDSession",
			"userIDSession", *userIDSession, "error", err.Error())
		return
	}

	err = app.Db.Get(category, stmt, category.Name, userID,
		time.Now().Format(revel.TimeFormats[0]), category.ID)
	if err != nil {
		log.Error("failed to update category", "category", category, "user ID", userID,
			"table", *table, "error", err.Error())
	}
	return
}

/*Delete a category from either faq_category or news_feed_category. */
func (category *Category) Delete(table *string) (err error) {

	stmt := `DELETE FROM ` + *table + `
		WHERE id = $1
	`

	_, err = app.Db.Exec(stmt, category.ID)
	if err != nil {
		log.Error("failed to delete category", "category", category,
			"table", *table, "error", err.Error())
	}
	return
}

/*Categories holds all categories of either the FAQs or the news feed. */
type Categories []Category

/*Select all categories and their respective entries from
either the FAQs or the NewsFeed table. */
func (categories *Categories) Select(table string) (err error) {

	stmt := `
		SELECT id, name, last_editor,
			TO_CHAR (last_edited AT TIME ZONE $1, 'YYYY-MM-DD HH24:MI:SS') as last_edited
		FROM ` + table + `
		ORDER BY name ASC
	`

	tx, err := app.Db.Beginx()
	if err != nil {
		log.Error("failed to begin tx", "error", err.Error())
		return err
	}

	err = tx.Select(categories, stmt, app.TimeZone)
	if err != nil {
		log.Error("failed to get categories", "table", table, "error", err.Error())
		tx.Rollback()
		return
	}

	stmt = stmtSelectFAQs
	if table != "faq_category" {
		stmt = stmtSelectNews
	}

	//get all entries of the current category
	for key := range *categories {
		err = tx.Select(&((*categories)[key].Entries), stmt, app.TimeZone, (*categories)[key].ID)
		if err != nil {
			log.Error("failed to get entries", "table", table, "category ID",
				(*categories)[key].ID, "error", err.Error())
			tx.Rollback()
			return
		}
	}

	tx.Commit()
	return
}

/*HelpPageEntry is a model for either a FAQ or a news feed entry. */
type HelpPageEntry struct {
	ID         int           `db:"id, primarykey, autoincrement"`
	CategoryID int           `db:"category_id"`
	LastEditor sql.NullInt32 `db:"last_editor"`
	LastEdited string        `db:"last_edited"`

	//determine the entry type
	IsFAQ bool

	//NewsFeed value
	Content string `db:"content"`

	//FAQ values
	Question string `db:"question"`
	Answer   string `db:"answer"`
}

/*Validate either FAQ or NewsFeed entry fields. */
func (entry *HelpPageEntry) Validate(v *revel.Validation) {

	v.Required(entry.CategoryID).
		MessageKey("validation.invalid.params")

	if entry.IsFAQ { //FAQ
		entry.Question = strings.TrimSpace(entry.Question)
		v.Check(entry.Question,
			revel.MinSize{3},
			revel.MaxSize{255},
		).MessageKey("validation.invalid.text.short")

		entry.Answer = strings.TrimSpace(entry.Answer)
		v.Check(entry.Answer,
			revel.MinSize{3},
			revel.MaxSize{255},
		).MessageKey("validation.invalid.text.short")

	} else { //news feed entry
		entry.Content = strings.TrimSpace(entry.Content)
		v.Check(entry.Content,
			revel.MinSize{3},
			revel.MaxSize{255},
		).MessageKey("validation.invalid.text.short")
	}
}

/*Insert a help page entry into either the faq or the news_feed table. */
func (entry *HelpPageEntry) Insert(userIDSession *string) (err error) {

	userID, err := strconv.Atoi(*userIDSession)
	if err != nil {
		log.Error("failed to parse userID from userIDSession",
			"userIDSession", *userIDSession, "error", err.Error())
		return
	}

	if entry.IsFAQ {
		err = app.Db.Get(entry, stmtInsertFAQ, entry.Question, entry.Answer, entry.CategoryID,
			userID, time.Now().Format(revel.TimeFormats[0]))
	} else {
		err = app.Db.Get(entry, stmtInsertNews, entry.Content, entry.CategoryID, userID,
			time.Now().Format(revel.TimeFormats[0]))
	}

	if err != nil {
		log.Error("failed to add entry", "entry", entry, "user ID", userID,
			"error", err.Error())
	}
	return
}

/*Update a help page entry in either the faq or the news_feed table. */
func (entry *HelpPageEntry) Update(userIDSession *string) (err error) {

	userID, err := strconv.Atoi(*userIDSession)
	if err != nil {
		log.Error("failed to parse userID from userIDSession",
			"userIDSession", *userIDSession, "error", err.Error())
		return
	}

	if entry.IsFAQ {
		err = app.Db.Get(entry, stmtUpdateFAQ, entry.Question, entry.Answer, entry.CategoryID,
			userID, time.Now().Format(revel.TimeFormats[0]), entry.ID)
	} else {
		err = app.Db.Get(entry, stmtUpdateNews, entry.Content, entry.CategoryID, userID,
			time.Now().Format(revel.TimeFormats[0]), entry.ID)
	}

	if err != nil {
		log.Error("failed to update entry", "entry", entry, "user ID", userID,
			"error", err.Error())
	}
	return
}

/*Delete an entry from either faq or news_feed. */
func (entry *HelpPageEntry) Delete(table *string) (err error) {

	stmt := `DELETE FROM ` + *table + `
		WHERE id = $1
	`

	_, err = app.Db.Exec(stmt, entry.ID)
	if err != nil {
		log.Error("failed to delete entry", "entry", entry,
			"table", *table, "error", err.Error())
	}
	return
}

/*HelpPageEntries holds all entries of a specified help page. */
type HelpPageEntries []HelpPageEntry

const (
	stmtSelectFAQs = `
		SELECT id, last_editor, question, answer, category_id,
			TO_CHAR (last_edited AT TIME ZONE $1, 'YYYY-MM-DD HH24:MI:SS') as last_edited
		FROM faqs
		WHERE category_id = $2
	`

	stmtSelectNews = `
		SELECT id, last_editor, content, category_id,
			TO_CHAR (last_edited AT TIME ZONE $1, 'YYYY-MM-DD HH24:MI:SS') as last_edited
		FROM news_feed
		WHERE category_id = $2
	`

	stmtInsertFAQ = `
		INSERT INTO faqs
			(question, answer, category_id, last_editor, last_edited)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, category_id
	`

	stmtInsertNews = `
		INSERT INTO news_feed
			(content, category_id, last_editor, last_edited)
		VALUES ($1, $2, $3, $4)
		RETURNING id, category_id
	`

	stmtUpdateFAQ = `
		UPDATE faqs
		SET question = $1, answer = $2, category_id = $3,
			last_editor = $4, last_edited = $5
		WHERE id = $6
		RETURNING id, category_id
	`

	stmtUpdateNews = `
		UPDATE news_feed
		SET content = $1, category_id = $2,
			last_editor = $3, last_edited = $4
		WHERE id = $5
		RETURNING id, category_id
	`
)
