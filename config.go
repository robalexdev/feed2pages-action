package main

import (
	"time"
)

type Config struct {
	FeedUrl string `yaml:"feed_url"`

	// Post limits and filters
	BlockWords       []string `yaml:"block_words"`
	BlockDomains     []string `yaml:"block_domains"`
	BlockPosts       []string `yaml:"block_posts"`
	PostAgeLimitDays *int     `yaml:"post_age_limit_days"`
	MaxPostsPerFeed  *int     `yaml:"max_posts_per_feed"`
	MaxPosts         *int     `yaml:"max_posts"`

	// Output folders
	ReadingFolderName   *string `yaml:"reading_folder_name"`
	FollowingFolderName *string `yaml:"following_folder_name"`
	DiscoverFolderName  *string `yaml:"discover_folder_name"`
	NetworkFolderName   *string `yaml:"network_folder_name"`

	// Discovery of recommended feeds
	DiscoverDepth             *int `yaml:"discover_depth"`
	MaxRecommendationsPerFeed *int `yaml:"max_recommendations_per_feed"`
	MaxRecommendations        *int `yaml:"max_recommendations"`
}

func strDefault(a *string, b string) string {
	if a != nil {
		return *a
	}
	return b
}

func intDefault(a *int, b int) int {
	if a != nil {
		return *a
	}
	return b
}

func (c *Config) Parse() *ParsedConfig {
	out := new(ParsedConfig)
	out.FeedUrl = c.FeedUrl
	out.BlockWords = c.BlockWords
	out.BlockDomains = c.BlockDomains

	out.BlockPosts = make(map[string]bool, len(c.BlockPosts))
	for _, blockTerm := range c.BlockPosts {
		out.BlockPosts[blockTerm] = true
	}

	out.ReadingFolderName = strDefault(c.ReadingFolderName, contentPath(DEFAULT_READING_FOLDER))
	out.FollowingFolderName = strDefault(c.FollowingFolderName, contentPath(DEFAULT_FOLLOWING_FOLDER))
	out.DiscoverFolderName = strDefault(c.DiscoverFolderName, contentPath(DEFAULT_DISCOVER_FOLDER))
	out.NetworkFolderName = strDefault(c.NetworkFolderName, contentPath(DEFAULT_NETWORK_FOLDER))

	ageLimit := -1 * intDefault(c.PostAgeLimitDays, 36500) // about 100 years ago
	out.PostAgeLimit = time.Now().AddDate(0, 0, ageLimit)

	out.MaxPosts = intDefault(c.MaxPosts, 1000)
	out.MaxPostsPerFeed = intDefault(c.MaxPostsPerFeed, 100)
	out.DiscoverDepth = intDefault(c.DiscoverDepth, 4)
	out.MaxRecommendations = intDefault(c.MaxRecommendations, 1000)
	out.MaxRecommendationsPerFeed = intDefault(c.MaxRecommendationsPerFeed, 100)

	return out
}

type ParsedConfig struct {
	FeedUrl string

	BlockWords   []string
	BlockDomains []string

	BlockPosts map[string]bool

	ReadingFolderName   string
	FollowingFolderName string
	DiscoverFolderName  string
	NetworkFolderName   string

	PostAgeLimit time.Time

	MaxPosts                  int
	MaxPostsPerFeed           int
	DiscoverDepth             int
	MaxRecommendations        int
	MaxRecommendationsPerFeed int
}
