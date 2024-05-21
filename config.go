package main

import (
	"net"
	"net/http"
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

	CrawlThreads   *int `yaml:"crawl_threads"`
	RequestTimeout *int `yaml:"request_timeout_ms"`

	// HTTP transport settings
	HttpDialKeepAlive         *int `yaml:"http_dial_keep_alive_ms"`
	HttpDialTimeout           *int `yaml:"http_dial_timeout_ms"`
	HttpExpectContinueTimeout *int `yaml:"http_expect_continue_timeout_ms"`
	HttpIdleConnTimeout       *int `yaml:"http_idle_conn_timeout_ms"`
	HttpTLSHandshakeTimeout   *int `yaml:"http_tls_handshake_timeout_ms"`
	HttpResponseHeaderTimeout *int `yaml:"http_response_header_timeout_ms"`

	HttpDialDualStack *bool `yaml:"http_dial_dual_stack"`
	HttpMaxIdleConns  *int  `yaml:"http_max_idle_conns"`
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

func durationDefaultNil(a_ms *int) *time.Duration {
	if a_ms != nil {
		r := time.Duration(*a_ms) * time.Millisecond
		return &r
	}
	return nil
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

	out.CrawlThreads = intDefault(c.CrawlThreads, 8)
	out.RequestTimeout = durationDefaultNil(c.RequestTimeout)

	out.HttpDialTimeout = durationDefaultNil(c.HttpDialTimeout)
	out.HttpDialKeepAlive = durationDefaultNil(c.HttpDialKeepAlive)
	out.HttpIdleConnTimeout = durationDefaultNil(c.HttpIdleConnTimeout)
	out.HttpTLSHandshakeTimeout = durationDefaultNil(c.HttpTLSHandshakeTimeout)
	out.HttpExpectContinueTimeout = durationDefaultNil(c.HttpExpectContinueTimeout)
	out.HttpResponseHeaderTimeout = durationDefaultNil(c.HttpResponseHeaderTimeout)

	out.HttpMaxIdleConns = c.HttpMaxIdleConns
	out.HttpDialDualStack = c.HttpDialDualStack

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

	CrawlThreads   int
	RequestTimeout *time.Duration

	HttpDialKeepAlive         *time.Duration
	HttpDialTimeout           *time.Duration
	HttpExpectContinueTimeout *time.Duration
	HttpIdleConnTimeout       *time.Duration
	HttpTLSHandshakeTimeout   *time.Duration
	HttpResponseHeaderTimeout *time.Duration

	HttpDialDualStack *bool
	HttpMaxIdleConns  *int
}

func (c *ParsedConfig) BuildTransport() *http.Transport {
	// Defaults: https://pkg.go.dev/net/http#DefaultTransport
	d := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	t := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           d.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	if c.HttpDialTimeout != nil {
		d.Timeout = *c.HttpDialTimeout
	}
	if c.HttpDialKeepAlive != nil {
		d.KeepAlive = *c.HttpDialKeepAlive
	}
	if c.HttpDialDualStack != nil {
		d.DualStack = *c.HttpDialDualStack
	}
	if c.HttpMaxIdleConns != nil {
		t.MaxIdleConns = *c.HttpMaxIdleConns
	}
	if c.HttpIdleConnTimeout != nil {
		t.IdleConnTimeout = *c.HttpIdleConnTimeout
	}
	if c.HttpTLSHandshakeTimeout != nil {
		t.TLSHandshakeTimeout = *c.HttpTLSHandshakeTimeout
	}
	if c.HttpExpectContinueTimeout != nil {
		t.ExpectContinueTimeout = *c.HttpExpectContinueTimeout
	}
	if c.HttpResponseHeaderTimeout != nil {
		t.ResponseHeaderTimeout = *c.HttpResponseHeaderTimeout
	}
	return t
}
