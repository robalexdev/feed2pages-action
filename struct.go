package main

import (
	"github.com/mmcdole/gofeed"
	"github.com/mmcdole/gofeed/rss"
)

type Config struct {
	Feeds   []FeedDetails
	FeedUrl string `yaml:"feed_url"`

	// Post limits and filters
	PostAgeLimitDays *int     `yaml:"post_age_limit_days"`
	MaxPostsPerFeed  *int     `yaml:"max_posts_per_feed"`
	MaxPosts         *int     `yaml:"max_posts"`
	BlockWords       []string `yaml:"block_words"`
	BlockDomains     []string `yaml:"block_domains"`
	BlockPosts       []string `yaml:"block_posts"`

	// Output folders
	ReadingFolderName   string `yaml:"reading_folder_name"`
	FollowingFolderName string `yaml:"following_folder_name"`
	DiscoverFolderName  string `yaml:"discover_folder_name"`

	// Discovery of recommended feeds
	DiscoverDepth             *int `yaml:"discover_depth"`
	MaxRecommendationsPerFeed *int `yaml:"max_recommendations_per_feed"`
	MaxRecommendations        *int `yaml:"max_recommendations"`
}

type FeedDetails struct {
	Link  string
	Text  string
	Title string
	Type  string
}

// https://gohugo.io/content-management/front-matter/
type PostFrontmatter struct {
	Date        string     `yaml:"date"`
	Params      PostParams `yaml:"params"`
	Title       string     `yaml:"title"`
	Description string     `yaml:"description"`
}

type FeedFrontmatter struct {
	Date        string     `yaml:"date"`
	Params      FeedParams `yaml:"params"`
	Title       string     `yaml:"title"`
	Description string     `yaml:"description"`
}

type DiscoverFrontmatter struct {
	Date        string         `yaml:"date"`
	Title       string         `yaml:"title"`
	Description string         `yaml:"description"`
	Params      DiscoverParams `yaml:"params"`
}

type PostParams struct {
	PrettyAge    string      `yaml:"pretty_age"`
	Post         gofeed.Item `yaml:"post"`
	FeedId       string      `yaml:"feed_id"`
	Feed         gofeed.Feed `yaml:"feed"`
	CommentsLink string      `yaml:"comments_link"`
}

type FeedParams struct {
	Id   string      `yaml:"id"`
	Feed gofeed.Feed `yaml:"feed"`
}

type DiscoverParams struct {
	Id            string   `yaml:"id"`
	Link          string   `yaml:"link"`
	RecommendedBy []string `yaml:"recommended_by"`
}

type PendingDiscover struct {
	RecommendedBy string
	Link          string
}

type MultiTypeItem struct {
	item *gofeed.Item
	rss  *rss.Item
}
