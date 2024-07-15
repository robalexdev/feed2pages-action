package main

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const fileName = "feed2pages.db"

type DB struct {
	db *gorm.DB
}

func NewDB() *DB {
	db := DB{}
	db.Open()
	db.Init()
	return &db
}

func (db *DB) Open() {
	var err error
	db.db, err = gorm.Open(sqlite.Open(fileName), &gorm.Config{
		PrepareStmt: true,
	})
	if err != nil {
		panicf("Can't open database: %v", err)
	}
}

func (db *DB) TrackBlogroll(fm *BlogrollFrontmatter) {
	blogroll := Blogroll{
		Date:        fm.Date,
		Description: fm.Description,
		Title:       fm.Title,
		BlogrollId:  fm.Params.Id,
		Link:        fm.Params.Link,
	}

	result := db.db.
		Clauses(
			clause.OnConflict{
				Columns: []clause.Column{{Name: "link"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"date", "description", "title",
				}),
			}).
		Create(&blogroll)
	ohno(result.Error)
}

func (db *DB) TrackFeed(fm *FeedFrontmatter) {
	feed := Feed{
		Date:        fm.Date,
		Description: fm.Description,
		Title:       fm.Title,
		FeedLink:    fm.Params.FeedLink,
		FeedId:      fm.Params.Id,
		FeedType:    fm.Params.FeedType,
		IsPodcast:   fm.Params.IsPodcast,
		IsNoarchive: fm.Params.IsNoarchive,
	}

	result := db.db.
		Clauses(
			clause.OnConflict{
				Columns: []clause.Column{{Name: "feed_link"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"date", "description", "title", "is_podcast", "is_noarchive",
				}),
			}).
		Create(&feed)
	ohno(result.Error)

	if len(fm.Params.Categories) > 0 {
		cats := []FeedsByCategory{}
		for _, cat := range fm.Params.Categories {
			cats = append(cats, FeedsByCategory{
				Category: cat,
				Link:     fm.Params.FeedLink,
			})
		}
		result = db.db.
			Clauses(clause.OnConflict{DoNothing: true}).
			Create(&cats)
		ohno(result.Error)
	}

	if len(fm.Params.Language) > 0 {
		result = db.db.
			Clauses(clause.OnConflict{DoNothing: true}).
			Create(&FeedsByLanguage{
				Language: fm.Params.Language,
				Link:     fm.Params.FeedLink,
			})
		ohno(result.Error)
	}
}

func (db *DB) TrackPost(fm *PostFrontmatter) {
	post := Post{
		Date:        fm.Date,
		Description: fm.Description,
		Title:       fm.Title,
		FeedId:      fm.Params.FeedId,
		PostLink:    fm.Params.Link,
		Guid:        fm.Params.Id,
	}

	result := db.db.
		Clauses(
			clause.OnConflict{
				Columns: []clause.Column{{Name: "guid"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"date", "description", "title", "post_link",
				}),
			}).
		Create(&post)
	ohno(result.Error)

	if len(fm.Params.Categories) > 0 {
		cats := []PostsByCategory{}
		for _, cat := range fm.Params.Categories {
			cats = append(cats, PostsByCategory{
				Category: cat,
				Link:     fm.Params.Link,
			})
		}
		result = db.db.
			Clauses(clause.OnConflict{DoNothing: true}).
			Create(&cats)
		ohno(result.Error)
	}

	if len(fm.Params.Language) > 0 {
		result = db.db.
			Clauses(clause.OnConflict{DoNothing: true}).
			Create(&PostsByLanguage{
				Language: fm.Params.Language,
				Link:     fm.Params.Link,
			})
		ohno(result.Error)
	}

	// TODO Link, blogrolls, recommended
}

func (db *DB) TrackLink(fm *LinkFrontmatter) {
	link := Link{
		SourceType:      int(fm.Params.SourceType),
		SourceUrl:       fm.Params.SourceURL,
		DestinationType: int(fm.Params.DestinationType),
		DestinationUrl:  fm.Params.DestinationURL,
		LinkType:        fm.Params.LinkType,
	}
	result := db.db.
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&link)
	ohno(result.Error)
}

func (db *DB) TrackNoIndex(link string) {
	noindex := Noindex{
		Link: link,
	}
	result := db.db.
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&noindex)
	ohno(result.Error)
}

func (db *DB) DeleteNoIndexLinks() {
	result := db.db.
		Where("source_url IN(?)",
			db.db.
				Model(&Noindex{}).
				Select("link")).
		Or(
			db.db.Where("source_url IN(?)",
				db.db.
					Model(&Noindex{}).
					Select("link"))).
		Delete(&Link{})
	ohno(result.Error)
	err := db.db.Migrator().DropTable("noindex")
	ohno(err)
}

type Blogroll struct {
	ID          uint   `gorm:"primaryKey"`
	Date        string // TODO: use time.Time
	Description string
	Title       string
	Link        string `gorm:"unique"`
	BlogrollId  string
}

type Feed struct {
	ID          uint   `gorm:"primaryKey"`
	Date        string // TODO: use time.Time
	Description string
	Title       string
	FeedLink    string `gorm:"unique"`
	FeedId      string
	FeedType    string
	IsPodcast   bool
	IsNoarchive bool
}

type Post struct {
	ID          uint   `gorm:"primaryKey"`
	Date        string // TODO: use time.Time
	Description string
	Title       string
	FeedId      string
	PostLink    string
	Guid        string `gorm:"unique"`
}

type PostsByCategory struct {
	ID       uint   `gorm:"primaryKey"`
	Category string `gorm:"uniqueIndex:uniquePostsByCat"`
	Link     string `gorm:"uniqueIndex:uniquePostsByCat"`
}

type FeedsByCategory struct {
	ID       uint   `gorm:"primaryKey"`
	Category string `gorm:"uniqueIndex:uniqueFeedsByCat"`
	Link     string `gorm:"uniqueIndex:uniqueFeedsByCat"`
}

type FeedsByLanguage struct {
	ID       uint   `gorm:"primaryKey"`
	Language string `gorm:"uniqueIndex:uniqueFeedsByLang"`
	Link     string `gorm:"uniqueIndex:uniqueFeedsByLang"`
}

type PostsByLanguage struct {
	ID       uint   `gorm:"primaryKey"`
	Language string `gorm:"uniqueIndex:uniquePostsByLang"`
	Link     string `gorm:"uniqueIndex:uniquePostsByLang"`
}

type Link struct {
	ID              uint   `gorm:"primaryKey"`
	SourceType      int    `gorm:"uniqueIndex:uniqueLink"`
	SourceUrl       string `gorm:"uniqueIndex:uniqueLink"`
	DestinationType int    `gorm:"uniqueIndex:uniqueLink"`
	DestinationUrl  string `gorm:"uniqueIndex:uniqueLink"`
	LinkType        string
}

type Noindex struct {
	ID   uint   `gorm:"primaryKey"`
	Link string `gorm:"uniqueIndex:uniqueNoindex"`
}

func (db *DB) Init() {
	db.db.AutoMigrate(&Link{})
	db.db.AutoMigrate(&Feed{})
	db.db.AutoMigrate(&Post{})
	db.db.AutoMigrate(&Blogroll{})
	db.db.AutoMigrate(&FeedsByCategory{})
	db.db.AutoMigrate(&FeedsByLanguage{})
	db.db.AutoMigrate(&PostsByCategory{})
	db.db.AutoMigrate(&PostsByLanguage{})
	db.db.AutoMigrate(&Noindex{})
}
