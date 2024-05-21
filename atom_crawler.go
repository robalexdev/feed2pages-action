package main

import (
	"github.com/antchfx/xmlquery"
	"github.com/gocolly/colly/v2"
	"log"
)

func (c *Crawler) OnXML_AtomFeed(r *colly.Request, channel *xmlquery.Node) {
	feed_url := r.URL.String()

	links := collectLinkHrefs("link[@rel='alternate']", channel)
	if len(links) == 0 {
		// If we can't find a rel=alternate link, fall back to all links
		links = collectLinkHrefs("link", channel)
	}

	title := xmlText(channel, "title")
	description := xmlText(channel, "subtitle")
	date := xmlText(channel, "updated")

	feed := NewFeedFrontmatter(feed_url)
	feed.WithDate(date)
	feed.WithTitle(title)
	feed.WithDescription(description)

	if blocked, blockWord := hasBlockWords(title, c.Config); blocked {
		log.Printf("Word in title is blocked: %s", blockWord)
		return
	}
	if blocked, blockWord := hasBlockWords(description, c.Config); blocked {
		log.Printf("Word in description is blocked: %s", blockWord)
		return
	}

	for _, link := range links {
		if isBlockedPost(link, title, feed.Params.Id, c.Config) {
			continue
		}

		if blocked, domain := isBlockedDomain(link, c.Config); blocked {
			log.Printf("Domain is blocked: %s", domain)
			continue
		}

		log.Println("DEPTH:", r.Depth)
		isDirect := r.Depth < 4
		feed.WithLink(link)
		feed.Save(isDirect, c.Config)

		log.Printf("Searching for blogroll in: %s", link)
		c.Request(NODE_TYPE_FEED, feed_url, NODE_TYPE_WEBSITE, link, r.Depth+1)
		recLink, err := buildRecommendationUrl(link)
		if err != nil {
			log.Println(err)
		} else {
			c.Request(NODE_TYPE_FEED, feed_url, NODE_TYPE_BLOGROLL, recLink, r.Depth+1)
		}
	}

	// Atom feeds don't have a blogroll syntax yet
	// Add here when they do
}

func (c *Crawler) OnXML_AtomEntry(r *colly.Request, entry *xmlquery.Node) {
	feed_url := r.URL.String()

	if r.Depth > 2 {
		return
	}

	post_id := xmlText(entry, "id")
	links := collectLinkHrefs("link[@rel='alternate']", entry)
	if len(links) == 0 {
		// If we can't find a rel=alt link, fall back to all links
		links = collectLinkHrefs("link", entry)
	}

	title := xmlText(entry, "title")
	date := xmlText(entry, "updated")
	content := xmlText(entry, "content")
	description := "" // Not supported

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

	for _, link := range links {
		post := NewPostFrontmatter(post_id, link)
		post.WithTitle(title)
		post.WithDescription(description)
		post.WithDate(date)
		post.WithContent(content)
		post.WithFeedLink(feed_url)

		if isBlockedPost(link, title, post.Params.Id, c.Config) {
			continue
		}
		if blocked, domain := isBlockedDomain(link, c.Config); blocked {
			log.Printf("Domain is blocked: %s", domain)
			continue
		}
		if title != "" {
			post.Save(c.Config)
		}
	}
}
