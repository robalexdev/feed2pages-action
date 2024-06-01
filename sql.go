package main

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

const fileName = "feed2pages.db"

type DB struct {
	db *sql.DB
}

func NewDB() *DB {
	db := DB{}
	db.Open()
	db.Init()
	return &db
}

func (db *DB) Open() {
	var err error
	db.db, err = sql.Open("sqlite3", fileName)
	if err != nil {
		panicf("Can't open database: %v", err)
	}
	db.db.SetMaxOpenConns(1)
}

func (db *DB) TrackFeed(fm *FeedFrontmatter) {
	_, err := db.db.Exec(`
    INSERT INTO feeds(date, description, title, feed_link, feed_id, feed_type)
      values(?,?,?,?,?, ?)
    ON CONFLICT(feed_link)
      DO UPDATE SET
        date=excluded.date,
        description=excluded.description,
        title=excluded.title;
    `,
		fm.Date,
		fm.Description,
		fm.Title,
		fm.Params.FeedLink,
		fm.Params.Id,
		fm.Params.FeedType,
	)

	if err != nil {
		panicf("Unable to track feed: %v", err)
	}

	for _, cat := range fm.Params.Categories {
		// TODO bulk inserts?
		db.TrackFeedCategory(cat, fm.Params.FeedLink)
	}
}

func (db *DB) TrackFeedCategory(cat, feedlink string) {
	_, err := db.db.Exec(`
    INSERT INTO feeds_by_categories(category, link)
      values(?,?)
    ON CONFLICT(category, link)
      DO NOTHING;
    `,
		cat,
		feedlink,
	)
	if err != nil {
		panicf("Unable to track feed: %v", err)
	}
}

func (db *DB) TrackPostCategory(cat, postlink string) {
	_, err := db.db.Exec(`
    INSERT INTO posts_by_categories(category, link)
      values(?,?)
    ON CONFLICT(category, link)
      DO NOTHING;
    `,
		cat,
		postlink,
	)

	if err != nil {
		panicf("Unable to track feed: %v", err)
	}
}

func (db *DB) TrackPost(fm *PostFrontmatter) {
	_, err := db.db.Exec(`
    INSERT INTO posts(date, description, title, feed_id, post_link, guid)
      values(?,?,?,?,?,?)
    ON CONFLICT(guid)
      DO UPDATE SET
        date=excluded.date,
        description=excluded.description,
        title=excluded.title,
        post_link=excluded.post_link;
    `,
		fm.Date,
		fm.Description,
		fm.Title,
		fm.Params.FeedId,
		fm.Params.Link,
		fm.Params.Id,
	)

	// TODO Link, blogrolls, recommended
	if err != nil {
		panicf("Unable to track link: %v", err)
	}

	for _, cat := range fm.Params.Categories {
		db.TrackPostCategory(cat, fm.Params.Link)
	}
}

func (db *DB) TrackLink(fm *LinkFrontmatter) {
	_, err := db.db.Exec(`
    INSERT INTO links(source_type, source_url, destination_type, destination_url, link_type)
      values(?,?,?,?,?)
    ON CONFLICT(source_type, source_url, destination_type, destination_url)
      DO NOTHING
    `,
		fm.Params.SourceType,
		fm.Params.SourceURL,
		fm.Params.DestinationType,
		fm.Params.DestinationURL,
		fm.Params.LinkType,
	)
	if err != nil {
		panicf("Unable to track link: %v", err)
	}
}

func (db *DB) TrackNoIndex(link string) {
	log.Printf("NOINDEX: %s\n", link)
	_, err := db.db.Exec(`INSERT INTO noindex(link) values(?)`, link)
	if err != nil {
		panicf("Unable to track noindex link: %v", err)
	}
}

func (db *DB) DeleteNoIndexLinks() {
	_, err := db.db.Exec(`
    DELETE FROM links
     WHERE source_url      IN(SELECT link FROM noindex)
        OR destination_url IN(SELECT link FROM noindex);
	`)
	if err != nil {
		panicf("Unable delete: %v", err)
	}
}

func (db *DB) Init() {
	query := `
  CREATE TABLE IF NOT EXISTS links (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_type INTEGER,
    source_url TEXT,
    destination_type INTEGER,
    destination_url TEXT,
    link_type TEXT,
    UNIQUE(source_type, source_url, destination_type, destination_url)
  );
  `
	var err error
	_, err = db.db.Exec(query)
	if err != nil {
		panicf("Unable to intialize database: %v", err)
	}

	query = `
  CREATE TABLE IF NOT EXISTS feeds (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date TEXT,
    description TEXT,
    title TEXT,
    feed_link TEXT,
    feed_id TEXT,
    feed_type TEXT,
    UNIQUE(feed_link)
  );
  `
	_, err = db.db.Exec(query)
	if err != nil {
		panicf("Unable to intialize database: %v", err)
	}

	query = `
  CREATE TABLE IF NOT EXISTS posts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date TEXT,
    description TEXT,
    title TEXT,
    feed_id TEXT,
    post_link TEXT,
    guid TEXT,
    UNIQUE(guid)
  );
  `
	_, err = db.db.Exec(query)
	if err != nil {
		panicf("Unable to intialize database: %v", err)
	}

	query = `
  CREATE TABLE IF NOT EXISTS feeds_by_categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    category TEXT,
    link TEXT,
    UNIQUE(category, link)
  );
  `
	_, err = db.db.Exec(query)
	if err != nil {
		panicf("Unable to intialize database: %v", err)
	}

	query = `
  CREATE TABLE IF NOT EXISTS posts_by_categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    category TEXT,
    link TEXT,
    UNIQUE(category, link)
  );
  `
	_, err = db.db.Exec(query)
	if err != nil {
		panicf("Unable to intialize database: %v", err)
	}

	query = `
  CREATE TEMP TABLE noindex (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    link TEXT
  );
  `
	_, err = db.db.Exec(query)
	if err != nil {
		panicf("Unable to intialize database: %v", err)
	}
}
