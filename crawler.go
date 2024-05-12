package main

import (
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/queue"
	"log"
	"net/http"
	"net/url"
	"os"
)

type Crawler struct {
	Collector *colly.Collector
	Config    *ParsedConfig
	Queue     *queue.Queue
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

func (c *Crawler) Request(u, recommender string, depth int) {
	parsed, err := url.Parse(u)
	if err != nil {
		return
	}

	if parsed.Scheme == "http" && parsed.Scheme == "https" {
		// prevent file:// and other schemes
		return
	}

	if blocked, domain := isBlockedDomain(u, c.Config); blocked {
		log.Printf("Skipping blocked domain: %d", domain)
		return
	}

	ctx := colly.NewContext()
	ctx.Put("rec", recommender)
	r := &colly.Request{
		URL:    parsed,
		Method: "GET",
		Depth:  depth,
		Ctx:    ctx,
	}
	c.Queue.AddRequest(r)
}

func (c *Crawler) OnResponseHandler(r *colly.Response) {
	// Save the recommendations link after it resolves
	recommender := r.Ctx.Get("rec")
	url := r.Request.URL.String()
	NewLinkFrontmatter(recommender, url).Save(c.Config)
}

func (c *Crawler) OnXML_OpmlOutline(outline *colly.XMLElement) {
	r := outline.Request
	blogroll_url := r.URL.String()
	feedUrl := outline.Attr("xmlUrl")
	if feedUrl == "" {
		return
	}
	feedUrl = outline.Request.AbsoluteURL(feedUrl)
	c.Request(feedUrl, blogroll_url, r.Depth+1)
}

func (c *Crawler) OnXML_RssChannel(channel *colly.XMLElement) {
	r := channel.Request
	feed_url := r.URL.String()

	link := channel.ChildText("/link")
	title := channel.ChildText("/title")
	description := channel.ChildText("/description")
	date := channel.ChildText("/pubDate")

	feed := NewFeedFrontmatter(feed_url)
	feed.WithDate(date)
	feed.WithTitle(title)
	feed.WithDescription(description)
	feed.WithLink(link)

	if blocked, domain := isBlockedDomain(link, c.Config); blocked {
		log.Printf("Domain is blocked: %s", domain)
		return
	}
	if blocked, blockWord := hasBlockWords(title, c.Config); blocked {
		log.Printf("Word in title is blocked: %s", blockWord)
		return
	}
	if blocked, blockWord := hasBlockWords(description, c.Config); blocked {
		log.Printf("Word in description is blocked: %s", blockWord)
		return
	}
	if isBlockedPost(link, title, feed.Params.Id, c.Config) {
		return
	}

	log.Println("DEPTH:", r.Depth)
	isDirect := r.Depth < 4
	feed.Save(isDirect, c.Config)

	// Check for blogrolls
	blogrolls := channel.ChildTexts("/source:blogroll")
	if len(blogrolls) != 0 {
		for _, blogroll := range blogrolls {
			log.Printf("Found blogroll: %s", blogroll)
			c.Request(blogroll, feed_url, r.Depth+1)
		}
	} else {
		log.Printf("Searching for blogroll in: %s", link)
		c.Request(link, feed_url, r.Depth+1)
		recLink, err := buildRecommendationUrl(link)
		if err != nil {
			log.Println(err)
		} else {
			c.Request(recLink, feed_url, r.Depth+1)
		}
	}
}

func (c *Crawler) OnXML_AtomFeed(channel *colly.XMLElement) {
	r := channel.Request
	feed_url := r.URL.String()

	link := channel.ChildAttr("/link[@rel='alternate']", "href")
	if link == "" {
		// Fallback link (any)
		link = channel.ChildAttr("/link", "href")
	}
	title := channel.ChildText("/title")
	description := channel.ChildText("/subtitle")
	date := channel.ChildText("/updated") // TODO: change format

	feed := NewFeedFrontmatter(feed_url)
	feed.WithDate(date)
	feed.WithTitle(title)
	feed.WithDescription(description)
	feed.WithLink(link)

	if blocked, domain := isBlockedDomain(link, c.Config); blocked {
		log.Printf("Domain is blocked: %s", domain)
		return
	}
	if blocked, blockWord := hasBlockWords(title, c.Config); blocked {
		log.Printf("Word in title is blocked: %s", blockWord)
		return
	}
	if blocked, blockWord := hasBlockWords(description, c.Config); blocked {
		log.Printf("Word in description is blocked: %s", blockWord)
		return
	}
	if isBlockedPost(link, title, feed.Params.Id, c.Config) {
		return
	}

	log.Println("DEPTH:", r.Depth)
	isDirect := r.Depth < 4
	feed.Save(isDirect, c.Config)

	// Atom feeds don't have a blogroll syntax yet
	// Add here when they do
}

func (c *Crawler) OnXML_RssItem(item *colly.XMLElement) {
	r := item.Request
	feed_url := r.URL.String()

	if r.Depth > 2 {
		return
	}

	post_id := item.ChildText("/guid")
	link := item.ChildText("/link")
	title := item.ChildText("/title")
	description := item.ChildText("/description")
	date := item.ChildText("/pubDate")
	content := item.ChildText("/content")

	post := NewPostFrontmatter(post_id, link)
	post.WithTitle(title)
	post.WithDescription(description)
	post.WithDate(date)
	post.WithContent(content)
	post.WithFeedLink(feed_url)

	if blocked, domain := isBlockedDomain(link, c.Config); blocked {
		log.Printf("Domain is blocked: %s", domain)
		return
	}
	if blocked, blockWord := hasBlockWords(title, c.Config); blocked {
		log.Printf("Word in title is blocked: %s", blockWord)
		return
	}
	if blocked, blockWord := hasBlockWords(description, c.Config); blocked {
		log.Printf("Word in description is blocked: %s", blockWord)
		return
	}
	if blocked, blockWord := hasBlockWords(content, c.Config); blocked {
		log.Printf("Word in content is blocked: %s", blockWord)
		return
	}
	if isBlockedPost(link, title, post.Params.Id, c.Config) {
		return
	}

	if title != "" {
		post.Save(c.Config)
	}
}

func (c *Crawler) OnXML_AtomEntry(item *colly.XMLElement) {
	r := item.Request
	feed_url := r.URL.String()

	if r.Depth > 2 {
		return
	}

	post_id := item.ChildText("/id")
	link := item.ChildAttr("/link[@rel='alternate']", "href")
	if link == "" {
		// Fallback link (any)
		link = item.ChildAttr("/link", "href")
	}
	title := item.ChildText("/title")
	date := item.ChildText("/updated")
	content := item.ChildText("/content")
	description := "" // Not supported

	post := NewPostFrontmatter(post_id, link)
	post.WithTitle(title)
	post.WithDescription(description)
	post.WithDate(date)
	post.WithContent(content)
	post.WithFeedLink(feed_url)

	if blocked, domain := isBlockedDomain(link, c.Config); blocked {
		log.Printf("Domain is blocked: %s", domain)
		return
	}
	if blocked, blockWord := hasBlockWords(title, c.Config); blocked {
		log.Printf("Word in title is blocked: %s", blockWord)
		return
	}
	if blocked, blockWord := hasBlockWords(description, c.Config); blocked {
		log.Printf("Word in description is blocked: %s", blockWord)
		return
	}
	if blocked, blockWord := hasBlockWords(content, c.Config); blocked {
		log.Printf("Word in content is blocked: %s", blockWord)
		return
	}
	if isBlockedPost(link, title, post.Params.Id, c.Config) {
		return
	}

	if title != "" {
		post.Save(c.Config)
	}
}

// Example:
// <link rel="blogroll" type="text/xml" href="https://feedland.com/opml?screenname=davewiner&catname=blogroll">
func (c *Crawler) OnHTML_RelLink(element *colly.HTMLElement) {
	r := element.Request
	page_url := r.URL.String()
	rel := element.Attr("rel")
	t := element.Attr("type")
	href := element.Attr("href")
	if href == "" {
		return
	}
	href = r.AbsoluteURL(href)
	if rel == "blogroll" && (t == "" || t == "text/xml" || t == "application/atom+xml") {
		log.Printf("Blogroll from HTML: %s", href)
		c.Request(href, page_url, r.Depth+1)
	}
}

func NewCrawler(config *ParsedConfig) Crawler {
	crawler := Crawler{}
	crawler.Config = config

	workingDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	t := &http.Transport{}
	t.RegisterProtocol("file", http.NewFileTransport(http.Dir(workingDir)))

	crawler.Collector = colly.NewCollector(
		colly.MaxDepth(config.DiscoverDepth),
		colly.UserAgent(USER_AGENT),
	)
	crawler.Collector.WithTransport(t)
	crawler.Collector.IgnoreRobotsTxt = false
	crawler.Collector.OnRequest(OnRequestHandler)
	crawler.Collector.OnResponse(crawler.OnResponseHandler)
	crawler.Collector.OnError(OnErrorHandler)

	// OPML blogroll
	crawler.Collector.OnXML("/opml/body//outline", crawler.OnXML_OpmlOutline)

	// RSS feed
	crawler.Collector.OnXML("/rss/channel", crawler.OnXML_RssChannel)
	crawler.Collector.OnXML("/rss/channel/item", crawler.OnXML_RssItem)

	// Atom feed
	crawler.Collector.OnXML("/feed", crawler.OnXML_AtomFeed)
	crawler.Collector.OnXML("/feed/entry", crawler.OnXML_AtomEntry)

	// HTML page
	crawler.Collector.OnHTML("link[rel='blogroll']", crawler.OnHTML_RelLink)

	crawler.Queue, err = queue.New(
		8,
		&queue.InMemoryQueueStorage{MaxSize: 10000},
	)
	if err != nil {
		panic(err)
	}

	return crawler
}
