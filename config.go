package main

import (
	"github.com/go-yaml/yaml"
	"net"
	"net/http"
	"slices"
	"time"
)

type NonOpmlBlogroll struct {
	Url      string `yaml:"url"`
	Handler  string `yaml:"handler"`
	Settings string `yaml:"settings"`
}


type Config struct {
	FeedUrls []string `yaml:"feed_urls"`
	NonOpmlBlogroll []NonOpmlBlogroll `yaml:"non_opml_blogroll_urls"`

	PrivateBlocksFile string `yaml:"private_blocks_file"`

	// Post limits and filters
	BlockWords       []string `yaml:"block_words"`
	BlockDomains     []string `yaml:"block_domains"`
	BlockPosts       []string `yaml:"block_posts"`
	PostAgeLimitDays *int     `yaml:"post_age_limit_days"`
	MaxPostsPerFeed  *int     `yaml:"max_posts_per_feed"`
	MaxPosts         *int     `yaml:"max_posts"`

	// Output modes
	OutputModes []string `yaml:"output_mode"`

	// Output folders
	ReadingFolderName   *string `yaml:"reading_folder_name"`
	FollowingFolderName *string `yaml:"following_folder_name"`
	DiscoverFolderName  *string `yaml:"discover_folder_name"`
	NetworkFolderName   *string `yaml:"network_folder_name"`
	BlogrollFolderName  *string `yaml:"blogroll_folder_name"`

	// Should we add content on-top-of existing content
	// or should we remove and replace it?
	RemoveOldContent *bool `yaml:"remove_old_content"`

	// Discovery of recommended feeds
	DiscoverDepth             *int `yaml:"discover_depth"`
	PostCollectionDepth       *int `yaml:"post_collection_depth"`
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

	// Proxy settings
	HttpProxyURL *string `yaml:"http_proxy_url"`

	HttpOnlyHosts []string `yaml:"http_only_hosts"`
}

type PrivateConfig struct {
	// Filters not shared with the world
	BlockWords   []string `yaml:"block_words"`
	BlockDomains []string `yaml:"block_domains"`
	BlockPosts   []string `yaml:"block_posts"`
}

func strDefault(a *string, b string) string {
	if a != nil {
		return *a
	}
	return b
}

func boolDefault(a *bool, b bool) bool {
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

func (c *Config) ParseOutputMode() []OutputMode {
	if len(c.OutputModes) == 0 {
		// Default to Hugo Content
		return []OutputMode{OUTPUT_MODE_HUGO_CONTENT}
	} else {
		for _, mode := range c.OutputModes {
			if !slices.Contains(OUTPUT_MODES, mode) {
				panicf("Unknown output mode: %s", mode)
			}
		}
		return c.OutputModes
	}
}

func (c *Config) Parse() *ParsedConfig {
	out := new(ParsedConfig)
	out.FeedUrls = c.FeedUrls
	out.BlockWords = c.BlockWords
	out.BlockDomains = c.BlockDomains

	for _, source := range c.NonOpmlBlogroll {
		if source.Handler == "jq" {
			for _, url := range jqProcessUrl(source.Url, source.Settings) {
				out.FeedUrls = append(out.FeedUrls, url)
			}
		}
	}
	// We'll likely have duplicates here
	out.FeedUrls = dedupeSlice(out.FeedUrls)

	out.BlockPosts = make(map[string]bool, len(c.BlockPosts))
	for _, blockTerm := range c.BlockPosts {
		out.BlockPosts[blockTerm] = true
	}

	if len(c.PrivateBlocksFile) > 0 {
		content, closer, err := readFile(c.PrivateBlocksFile)
		if err != nil {
			panicf("Unable to parse config: %e", err)
		}
		defer closer.Close()
		priv := PrivateConfig{}
		decoder := yaml.NewDecoder(content)
		err = decoder.Decode(&priv)
		ohno(err)

		out.BlockWords = append(out.BlockWords, priv.BlockWords...)
		out.BlockDomains = append(out.BlockDomains, priv.BlockDomains...)
		for _, blockTerm := range priv.BlockPosts {
			out.BlockPosts[blockTerm] = true
		}
	}

	out.OutputModes = c.ParseOutputMode()

	out.ReadingFolderName = strDefault(c.ReadingFolderName, contentPath(DEFAULT_READING_FOLDER))
	out.FollowingFolderName = strDefault(c.FollowingFolderName, contentPath(DEFAULT_FOLLOWING_FOLDER))
	out.DiscoverFolderName = strDefault(c.DiscoverFolderName, contentPath(DEFAULT_DISCOVER_FOLDER))
	out.NetworkFolderName = strDefault(c.NetworkFolderName, contentPath(DEFAULT_NETWORK_FOLDER))
	out.BlogrollFolderName = strDefault(c.BlogrollFolderName, contentPath(DEFAULT_BLOGROLL_FOLDER))

	out.RemoveOldContent = boolDefault(c.RemoveOldContent, true)

	ageLimit := -1 * intDefault(c.PostAgeLimitDays, 36500) // about 100 years ago
	out.PostAgeLimit = time.Now().AddDate(0, 0, ageLimit)

	out.MaxPosts = intDefault(c.MaxPosts, 1000)
	out.MaxPostsPerFeed = intDefault(c.MaxPostsPerFeed, 100)
	out.DiscoverDepth = intDefault(c.DiscoverDepth, 4)
	out.PostCollectionDepth = intDefault(c.PostCollectionDepth, 2)
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

	out.HttpProxyURL = c.HttpProxyURL

	out.HttpOnlyHosts = c.HttpOnlyHosts

	return out
}

type ParsedConfig struct {
	FeedUrls []string


	BlockWords   []string
	BlockDomains []string

	BlockPosts map[string]bool

	OutputModes []OutputMode

	ReadingFolderName   string
	FollowingFolderName string
	DiscoverFolderName  string
	NetworkFolderName   string
	BlogrollFolderName  string

	RemoveOldContent bool

	PostAgeLimit time.Time

	MaxPosts                  int
	MaxPostsPerFeed           int
	DiscoverDepth             int
	PostCollectionDepth       int
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

	HttpProxyURL  *string
	HttpOnlyHosts []string
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
