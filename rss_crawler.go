package main

import (
	"github.com/antchfx/xmlquery"
	"github.com/gocolly/colly/v2"
	"log"
)

func (c *Crawler) OnXML_RssChannel(r *colly.Request, channel *xmlquery.Node) {
	feed_url := r.URL.String()

	link := xmlText(channel, "link")
	title := xmlText(channel, "title")
	description := xmlText(channel, "description")
	date := xmlText(channel, "pubDate")

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
	feed.WithBlogRolls(blogrollUrls)

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
	for _, blogroll := range blogrollUrls {
		log.Printf("Found blogroll: %s", blogroll)
		c.Request(NODE_TYPE_FEED, feed_url, NODE_TYPE_BLOGROLL, blogroll, r.Depth+1)
	}

	if link != "" {
		log.Printf("Searching for blogroll in: %s", link)
		c.Request(NODE_TYPE_FEED, feed_url, NODE_TYPE_WEBSITE, link, r.Depth+1)
		recLink, err := buildRecommendationUrl(link)
		if err != nil {
			log.Println(err)
		} else {
			c.Request(NODE_TYPE_FEED, feed_url, NODE_TYPE_BLOGROLL, recLink, r.Depth+1)
		}
	}
}

func (c *Crawler) OnXML_RssItem(r *colly.Request, item *xmlquery.Node) {
	feed_url := r.URL.String()

	if r.Depth > 2 {
		return
	}

	post_id := xmlText(item, "guid")
	link := xmlText(item, "link")
	title := xmlText(item, "title")
	description := xmlText(item, "description")
	date := xmlText(item, "pubDate")
	content := xmlText(item, "content")

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
