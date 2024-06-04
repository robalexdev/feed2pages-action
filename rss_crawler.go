package main

import (
	"cmp"
	"github.com/antchfx/xmlquery"
	"github.com/gocolly/colly/v2"
	"log"
	"slices"
)

func (c *Crawler) OnXML_RssChannel(r *colly.Request, channel *xmlquery.Node) {
	isPodcast := false
	feed_url := r.URL.String()

	link := xmlText(channel, "link")
	title := xmlText(channel, "title")
	description := xmlText(channel, "description")
	date := fmtDate(xmlText(channel, "pubDate"))

	// Podcasts may use iTunes categories
	categories := xmlTextMultipleWithNamespace(channel, c.ITunesCategoryWithNamespaceXPath)
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
	feed.IsPodcast(isPodcast)

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

	c.SaveFeed(feed, isDirect)

	// Check for blogrolls
	for _, blogroll := range blogrollUrls {
		log.Printf("Found blogroll: %s", blogroll)
		c.Request(NODE_TYPE_FEED, feed_url, NODE_TYPE_BLOGROLL, blogroll, LINK_TYPE_FROM_FEED, r.Depth+1)
	}

	if link != "" {
		log.Printf("Searching for blogroll in: %s", link)
		c.Request(NODE_TYPE_FEED, feed_url, NODE_TYPE_WEBSITE, link, LINK_TYPE_FROM_FEED, r.Depth+1)
	}

	c.CollectRssItems(r, channel)
}

func (c *Crawler) CollectRssItems(r *colly.Request, channel *xmlquery.Node) {
	if r.Depth > c.Config.PostCollectionDepth {
		return
	}
	if c.Config.MaxPostsPerFeed < 1 {
		return
	}

	posts := []*PostFrontmatter{}
	xmlItems := xmlquery.Find(channel, "//item")
	for _, item := range xmlItems {
		post, ok := c.OnXML_RssItem(r, item)
		if ok {
			posts = append(posts, post)
		}
	}

	slices.SortFunc(posts, func(a, b *PostFrontmatter) int {
		// Reverse chronological
		return cmp.Compare(b.Date, a.Date)
	})

	for i, post := range posts {
		if i < c.Config.MaxPostsPerFeed {
			c.SavePost(post)
		}
	}
}

func (c *Crawler) OnXML_RssItem(r *colly.Request, item *xmlquery.Node) (*PostFrontmatter, bool) {
	feed_url := r.URL.String()

	post_id := xmlText(item, "guid")
	link := xmlText(item, "link")
	title := xmlText(item, "title")
	description := xmlText(item, "description")
	date := fmtDate(xmlText(item, "pubDate"))
	content := xmlText(item, "content")
	categories := xmlTextMultiple(item, "category")

	post := NewPostFrontmatter(post_id, link)
	post.WithTitle(title)
	post.WithDescription(description)
	post.WithDate(date)
	post.WithContent(content)
	post.WithFeedLink(feed_url)
	post.WithCategories(categories)

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

	return post, true
}
