package main

import (
	"github.com/antchfx/xmlquery"
	"github.com/gocolly/colly/v2"
	"log"
	"net/http"
	"slices"
	"strings"
)

func (c *Crawler) OnXML_RssChannel(headers *http.Header, r *colly.Request, channel *xmlquery.Node) {
	isPodcast := false
	feed_url := r.URL.String()

	link := xmlText(channel, "link[not(@rel=\"next\")]")
	title := xmlText(channel, "title")
	description := xmlText(channel, "description")
	date := fmtDate(xmlText(channel, "pubDate"))
	language := xmlText(channel, "language")

	// Podcasts may use iTunes categories
	categories := xmlPathAttrMultipleWithNamespace(channel, c.ITunesCategoryWithNamespaceXPath, "text")
	if len(categories) > 0 {
		// iTunes requires this for a podcast to be listed
		isPodcast = true
	} else {
		categories = xmlTextMultiple(channel, "category")
	}

	// First try a namespace aware query for blogroll
	blogrolls := xmlquery.QuerySelectorAll(channel, c.BlogrollWithNamespaceXPath)
	if len(blogrolls) == 0 {
		// Then fallback
		blogrolls = xmlquery.Find(channel, "blogroll")
	}

	blogrollUrls := []string{}
	for _, node := range blogrolls {
		found := xmlText(node, "text()")
		blogrollUrls = append(blogrollUrls, found)
	}

	feed := NewFeedFrontmatter(feed_url)
	feed.WithDate(date)
	feed.WithTitle(title)
	feed.WithDescription(description)
	feed.WithLink(link)
	feed.WithFeedType("rss")
	feed.WithBlogRolls(blogrollUrls)
	feed.WithCategories(categories)
	feed.WithLanguage(language)
	feed.IsPodcast(isPodcast)
	setNoArchive(feed, headers)

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

	// Check for blogrolls
	for _, blogroll := range blogrollUrls {
		log.Printf("Found blogroll: %s", blogroll)
		c.Request(NODE_TYPE_FEED, feed_url, NODE_TYPE_BLOGROLL, blogroll, LINK_TYPE_FROM_FEED, r.Depth+1)
	}

	if link != "" {
		log.Printf("Searching for blogroll in: %s", link)
		c.Request(NODE_TYPE_FEED, feed_url, NODE_TYPE_WEBSITE, link, LINK_TYPE_FROM_FEED, r.Depth+1)
	}

	postCount, avgPostLen, avgPostPerDay := c.CollectRssItems(r, channel, language)
	feed.WithPostCount(postCount)
	feed.WithAvgPostLen(avgPostLen)
	feed.WithAvgPostPerDay(avgPostPerDay)
	c.SaveFeed(feed, isDirect)
}

func (c *Crawler) CollectRssItems(r *colly.Request, channel *xmlquery.Node, feed_language string) (int, int, float32) {
	if r.Depth > c.Config.PostCollectionDepth {
		return 0, 0, 0.0
	}
	if c.Config.MaxPostsPerFeed < 1 {
		return 0, 0, 0.0
	}

	posts := []*PostFrontmatter{}
	xmlItems := xmlquery.Find(channel, "//item")

	for _, item := range xmlItems {
		post, ok := c.OnXML_RssItem(r, item, feed_language)
		if ok {
			posts = append(posts, post)
		}
	}

	slices.SortFunc(posts, func(a, b *PostFrontmatter) int {
		// Reverse chronological
		return cmpDateStr(b.Date, a.Date)
	})

	// TODO remove posts from the future

	postLenSum := 0
	for i, post := range posts {
		postLenSum += len(post.Params.Content)
		if i < c.Config.MaxPostsPerFeed {
			c.SavePost(post)
		}
	}

	numPosts := len(posts)
	avgPostLen := 0
	avgPostPerDay := float32(0.0)
	if numPosts > 0 {
		avgPostLen = int(postLenSum / numPosts)
	}
	if numPosts >= 2 {
		newestDate, err := ParseDate(posts[0].Date)
		if err == nil {
			oldestDate, err := ParseDate(posts[len(posts)-1].Date)
			if err == nil {
				durationNs := float32(newestDate.Sub(oldestDate))
				durationDays := durationNs / 1000 / 1000 / 60 / 60 / 24
				if durationDays > 0 {
					avgPostPerDay = float32(numPosts) / durationDays
				}
			}
		}
	}

	// TODO: Additional stats: oldest post, newest pos
	// TODO: lastActive
	return numPosts, avgPostLen, avgPostPerDay
}

func (c *Crawler) OnXML_RssItem(r *colly.Request, item *xmlquery.Node, feed_language string) (*PostFrontmatter, bool) {
	feed_url := r.URL.String()

	post_id := xmlText(item, "guid")
	link := xmlText(item, "link")
	title := xmlText(item, "title")
	description := xmlText(item, "description")
	date := fmtDate(xmlText(item, "pubDate"))
	content := xmlText(item, "content")
	categories := xmlTextMultiple(item, "category")

	post := NewPostFrontmatter(feed_url, post_id, link)
	post.WithTitle(title)
	post.WithDescription(description)
	post.WithDate(date)
	post.WithContent(content)
	post.WithFeedLink(feed_url)
	post.WithCategories(categories)

	// TODO: Should we try xml:lang too?
	post.WithLanguage(feed_language)

	if title == "" {
		return nil, false
	}
	if blocked, domain := isBlockedDomain(link, c.Config); blocked {
		log.Printf("Domain is blocked: %s", domain)
		return nil, false
	}
	if blocked, blockWord := hasBlockWords(title, c.Config); blocked {
		log.Printf("Word in title is blocked: %s", blockWord)
		return nil, false
	}
	if blocked, blockWord := hasBlockWords(description, c.Config); blocked {
		log.Printf("Word in description is blocked: %s", blockWord)
		return nil, false
	}
	if blocked, blockWord := hasBlockWords(content, c.Config); blocked {
		log.Printf("Word in content is blocked: %s", blockWord)
		return nil, false
	}
	if isBlockedPost(link, title, post.Params.Id, c.Config) {
		return nil, false
	}
	if strings.HasPrefix(link, "/") {
		// This is a relative URL which are not well supported by readers
		return nil, false
	}
	if !isWebLink(link) {
		// This isn't a web link
		return nil, false
	}

	return post, true
}
