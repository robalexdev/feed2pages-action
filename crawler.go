package main

import (
	"bytes"
	"github.com/antchfx/xmlquery"
	"github.com/antchfx/xpath"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/queue"
	"log"
	"net/http"
	"net/url"
	"os"
)

type Crawler struct {
	Collector                  *colly.Collector
	Config                     *ParsedConfig
	Queue                      *queue.Queue
	BlogrollWithNamespaceXPath *xpath.Expr
}

func OnErrorHandler(resp *colly.Response, err error) {
	log.Printf("Crawl error: %s, %v", resp.Request.URL, err)
}

func OnRequestHandler(r *colly.Request) {
	url := r.URL.String()
	log.Printf("Processing request: %s", url)
}

func (c *Crawler) Crawl(urls ...string) error {
	for _, url := range urls {
		c.Collector.Visit(url)
	}
	return c.Queue.Run(c.Collector)
}

func (c *Crawler) Request(recommender_type NodeType, recommender string, target_type NodeType, target string, depth int) {
	parsed, err := url.Parse(target)
	if err != nil {
		return
	}

	if parsed.Scheme == "http" && parsed.Scheme == "https" {
		// prevent file:// and other schemes
		return
	}

	if blocked, domain := isBlockedDomain(target, c.Config); blocked {
		log.Printf("Skipping blocked domain: %d", domain)
		return
	}

	if target_type == NODE_TYPE_BLOGROLL {
		// Special case:
		// When we find a website we guess that it may have a blogroll at the well-known
		// URL. But we don't want to track it until we load it
	} else {
		// Others are tracked immediately as someone thought the URL was valid enough
		// to link to it.
		NewLinkFrontmatter(recommender_type, recommender, target_type, target).Save(c.Config)
	}

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

func (c *Crawler) SaveFrontmatter(request *colly.Request) {
	url := request.URL.String()
	recommender := request.Ctx.Get("rec")
	ctx_recommender_type := request.Ctx.GetAny("rec_type")
	ctx_target_type := request.Ctx.GetAny("target_type")

	recommender_type := NODE_TYPE_SEED
	if ctx_recommender_type != nil {
		recommender_type = ctx_recommender_type.(NodeType)
	}
	target_type := NODE_TYPE_FEED
	if ctx_target_type != nil {
		target_type = ctx_target_type.(NodeType)
	}

	NewLinkFrontmatter(recommender_type, recommender, target_type, url).Save(c.Config)
}

func processXmlQuery(r *colly.Request, xpathStr string, nav *xmlquery.Node, callback func(*colly.Request, *xmlquery.Node)) {
	foundNodes := xmlquery.Find(nav, xpathStr)
	for _, found := range foundNodes {
		callback(r, found)
	}
}

func (c *Crawler) OnResponseHandler(resp *colly.Response) {
	r := resp.Request
	if resp.StatusCode != 200 {
		return
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

	processXmlQuery(r, "/opml/body", doc, c.OnXML_OpmlBody)
	processXmlQuery(r, "/opml/body//outline", doc, c.OnXML_OpmlOutline)
	processXmlQuery(r, "/rss/channel", doc, c.OnXML_RssChannel)
	processXmlQuery(r, "/rss/channel/item", doc, c.OnXML_RssItem)
	processXmlQuery(r, "/feed", doc, c.OnXML_AtomFeed)
	processXmlQuery(r, "/feed/entry", doc, c.OnXML_AtomEntry)
}

func NewCrawler(config *ParsedConfig) Crawler {
	crawler := Crawler{}
	crawler.Config = config

	var err error
	nsMap := map[string]string{
		"source": "http://source.scripting.com/",
	}
	crawler.BlogrollWithNamespaceXPath, err = xpath.CompileWithNS("source:blogroll", nsMap)
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

	t := config.BuildTransport()
	t.RegisterProtocol("file", http.NewFileTransport(http.Dir(workingDir)))
	crawler.Collector.WithTransport(t)
	crawler.Collector.DisableCookies()

	crawler.Collector.IgnoreRobotsTxt = false
	if config.RequestTimeout != nil {
		crawler.Collector.SetRequestTimeout(*config.RequestTimeout)
	}
	crawler.Collector.OnRequest(OnRequestHandler)
	crawler.Collector.OnError(OnErrorHandler)

	// XML handled here: OPML, RSS, Atom
	crawler.Collector.OnResponse(crawler.OnResponseHandler)

	// HTML pages
	crawler.Collector.OnHTML("link[rel='blogroll']", crawler.OnHTML_RelLink)

	crawler.Queue, err = queue.New(
		config.CrawlThreads,
		&queue.InMemoryQueueStorage{MaxSize: 10000},
	)
	if err != nil {
		panic(err)
	}

	return crawler
}
