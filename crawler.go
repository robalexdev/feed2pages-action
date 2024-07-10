package main

import (
	"bytes"
	"github.com/antchfx/xmlquery"
	"github.com/antchfx/xpath"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/queue"
	"github.com/goware/urlx"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
)

type Crawler struct {
	Collector                        *colly.Collector
	Config                           *ParsedConfig
	Queue                            *queue.Queue
	BlogrollWithNamespaceXPath       *xpath.Expr
	ITunesCategoryWithNamespaceXPath *xpath.Expr
	db                               *DB
}

func (c *Crawler) OnErrorHandler(resp *colly.Response, err error) {
	// Don't index certain HTTP error responses
	noIndexStatusCodes := []int{
		401, // Unauthorized
		403, // Forbidden
		404, // Not found
		405, // Method not allowed
		407, // Proxy auth. req.
		410, // Gone
	}
	for code := range noIndexStatusCodes {
		if resp.StatusCode == code {
			c.db.TrackNoIndex(resp.Request.URL.String())
			break
		}
	}

	log.Printf("Crawl error: %s, %v %v", resp.Request.URL, resp.StatusCode, err)
}

func OnRequestHandler(r *colly.Request) {
	url := r.URL.String()
	log.Printf("Processing request: %s", url)
	r.Headers.Set("Referer", REFERER_STRING)
}

func (c *Crawler) Crawl(urls ...string) {
	for _, url := range urls {
		c.Collector.Visit(url)
	}
	err := c.Queue.Run(c.Collector)
	if err != nil {
		panicf("Config decode error: %e", err)
	}
}

func (c *Crawler) PurgeNoIndex() {
	c.db.DeleteNoIndexLinks()
}

func (c *Crawler) Request(recommender_type NodeType, recommender string, target_type NodeType, target string, link_type string, depth int) {
	// Common parsing issue
	if strings.HasPrefix(target, "mailto:") {
		return
	}

	parsed, err := urlx.Parse(target)
	if err != nil {
		return
	}

	// Upgrade HTTP to HTTPS
	if parsed.Scheme == "http" && !slices.Contains(c.Config.HttpOnlyHosts, parsed.Host) {
		parsed.Scheme = "https"
	}

	// Normalize
	if parsed.Path == "" {
		parsed.Path = "/"
	}

	// prevent file:// and other schemes
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return
	}

	// Normalize URL
	target, err = urlx.Normalize(parsed)
	if err != nil {
		return
	}

	if blocked, domain := isBlockedDomain(target, c.Config); blocked {
		log.Printf("Skipping blocked domain: %d", domain)
		return
	}

	// Record a link between the recommender URL and this one
	link := NewLinkFrontmatter(recommender_type, recommender, target_type, target, link_type)
	c.SaveLink(link)

	ctx := colly.NewContext()
	ctx.Put("rec", recommender)
	ctx.Put("rec_type", recommender_type)
	ctx.Put("target_type", target_type)
	r := &colly.Request{
		URL:    parsed,
		Method: "GET",
		Depth:  depth,
		Ctx:    ctx,
	}
	c.Queue.AddRequest(r)
}

func processXmlQuery(headers *http.Header, r *colly.Request, xpathStr string, nav *xmlquery.Node, callback func(*http.Header, *colly.Request, *xmlquery.Node)) {
	foundNodes := xmlquery.Find(nav, xpathStr)
	for _, found := range foundNodes {
		callback(headers, r, found)
	}
}

func (c *Crawler) OnResponseHandler(resp *colly.Response) {
	r := resp.Request
	page_url := r.URL.String()
	if resp.StatusCode != 200 {
		return
	}

	headers := resp.Headers
	if headers != nil {
		for _, headerVal := range resp.Headers.Values("X-Robots-Tag") {
			if ContainsAnyString(headerVal, META_ROBOT_NOINDEX_VARIANTS) {
				c.db.TrackNoIndex(page_url)
				return
			}
		}
	}

	opts := xmlquery.ParserOptions{
		Decoder: &xmlquery.DecoderOptions{
			Strict: false,
		},
	}
	doc, err := xmlquery.ParseWithOptions(bytes.NewBuffer(resp.Body), opts)
	if err != nil {
		log.Printf("Unable to parse as XML for %s: %v", r.URL.String(), err)
		return
	}

	processXmlQuery(headers, r, "/opml/body//outline", doc, c.OnXML_OpmlOutline)
	processXmlQuery(headers, r, "/rss/channel", doc, c.OnXML_RssChannel)
	processXmlQuery(headers, r, "/feed", doc, c.OnXML_AtomFeed)
}

func NewCrawler(config *ParsedConfig) Crawler {
	crawler := Crawler{}
	crawler.Config = config
	crawler.db = NewDB()

	var err error
	nsMap := map[string]string{
		"source": "http://source.scripting.com/",
		"itunes": "http://www.itunes.com/dtds/podcast-1.0.dtd",
	}
	crawler.BlogrollWithNamespaceXPath, err = xpath.CompileWithNS("source:blogroll", nsMap)
	if err != nil {
		panic(err)
	}

	crawler.ITunesCategoryWithNamespaceXPath, err = xpath.CompileWithNS("itunes:category", nsMap)
	if err != nil {
		panic(err)
	}

	workingDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	crawler.Collector = colly.NewCollector(
		colly.MaxDepth(config.DiscoverDepth),
		colly.UserAgent(USER_AGENT),
	)
	crawler.Collector.CacheDir = "./cache"

	t := config.BuildTransport()
	t.RegisterProtocol("file", http.NewFileTransport(http.Dir(workingDir)))
	crawler.Collector.WithTransport(t)
	crawler.Collector.DisableCookies()
	if config.HttpProxyURL != nil {
		crawler.Collector.SetProxy(*config.HttpProxyURL)
	}

	crawler.Collector.IgnoreRobotsTxt = false
	if config.RequestTimeout != nil {
		crawler.Collector.SetRequestTimeout(*config.RequestTimeout)
	}
	crawler.Collector.OnRequest(OnRequestHandler)
	crawler.Collector.OnError(crawler.OnErrorHandler)

	// XML handled here: OPML, RSS, Atom
	crawler.Collector.OnResponse(crawler.OnResponseHandler)

	// HTML pages
	crawler.Collector.OnHTML("html", crawler.OnHTML)

	crawler.Queue, err = queue.New(
		config.CrawlThreads,
		&queue.InMemoryQueueStorage{MaxSize: 10000},
	)
	if err != nil {
		panic(err)
	}

	return crawler
}

func (c *Crawler) SaveLink(f *LinkFrontmatter) {
	if slices.Contains(c.Config.OutputModes, OUTPUT_MODE_HUGO_CONTENT) {
		id := buildLinkId(f.Params.SourceURL, f.Params.DestinationURL)
		path := generatedFilePath(c.Config.NetworkFolderName, LINK_PREFIX, id)
		writeYaml(f, path)
	}
	if slices.Contains(c.Config.OutputModes, OUTPUT_MODE_SQL) {
		c.db.TrackLink(f)
	}
}

func (c *Crawler) SaveFeed(f *FeedFrontmatter, isDirect bool) {
	if slices.Contains(c.Config.OutputModes, OUTPUT_MODE_HUGO_CONTENT) {
		var path string
		if isDirect {
			path = generatedFilePath(c.Config.FollowingFolderName, FEED_PREFIX, f.Params.Id)
		} else {
			path = generatedFilePath(c.Config.DiscoverFolderName, FEED_PREFIX, f.Params.Id)
		}
		writeYaml(f, path)
	}
	if slices.Contains(c.Config.OutputModes, OUTPUT_MODE_SQL) {
		c.db.TrackFeed(f)
	}
}

func (c *Crawler) SavePost(f *PostFrontmatter) {
	if slices.Contains(c.Config.OutputModes, OUTPUT_MODE_HUGO_CONTENT) {
		path := generatedFilePath(c.Config.ReadingFolderName, POST_PREFIX, f.Params.Id)
		writeYaml(f, path)
	}
	if slices.Contains(c.Config.OutputModes, OUTPUT_MODE_SQL) {
		c.db.TrackPost(f)
	}
}
