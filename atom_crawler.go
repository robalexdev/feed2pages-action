package main

import (
	"github.com/antchfx/xmlquery"
	"github.com/gocolly/colly/v2"
	"log"
	"net/http"
	"slices"
	"strings"
)

func (c *Crawler) OnXML_AtomFeed(headers *http.Header, r *colly.Request, channel *xmlquery.Node) {
	feed_url := r.URL.String()
	links := collectLinkHrefs(r, "link[@rel='alternate']", channel)
	title := xmlText(channel, "title")
	description := xmlText(channel, "subtitle")
	date := fmtDate(xmlText(channel, "updated"))
	categories := xmlPathAttrMultiple(channel, "category", "term")

	// Find a top level language
	language := strings.TrimSpace(channel.SelectAttr("xml:lang"))

	feed := NewFeedFrontmatter(feed_url)
	feed.WithDate(date)
	feed.WithTitle(title)
	feed.WithFeedType("atom")
	feed.WithDescription(description)
	feed.WithCategories(categories)
	feed.WithLanguage(language)
	setNoArchive(feed, headers)

	if blocked, blockWord := hasBlockWords(title, c.Config); blocked {
		log.Printf("Word in title is blocked: %s", blockWord)
		return
	}
	if blocked, blockWord := hasBlockWords(description, c.Config); blocked {
		log.Printf("Word in description is blocked: %s", blockWord)
		return
	}

	link := ""
	if len(links) > 0 {
		link = links[0]
		feed.WithLink(link)
		if len(links) > 1 {
			log.Printf("TODO: Add support for multiple links")
		}
	}

	if isBlockedPost(link, title, feed.Params.Id, c.Config) {
		return
	}

	if blocked, domain := isBlockedDomain(link, c.Config); blocked {
		log.Printf("Domain is blocked: %s", domain)
		return
	}

	log.Println("DEPTH:", r.Depth)
	isDirect := r.Depth < 4
	c.SaveFeed(feed, isDirect)

	if len(link) > 0 {
		log.Printf("Searching for blogroll in: %s", link)
		c.Request(NODE_TYPE_FEED, feed_url, NODE_TYPE_WEBSITE, link, LINK_TYPE_LINK_REL_ALT, r.Depth+1)
	}

	// Atom feeds don't have a blogroll syntax yet
	// Add here when they do

	c.CollectAtomEntries(r, channel, language)
}

func (c *Crawler) CollectAtomEntries(r *colly.Request, channel *xmlquery.Node, feed_language string) {
	if r.Depth > c.Config.PostCollectionDepth {
		return
	}
	if c.Config.MaxPostsPerFeed < 1 {
		return
	}

	posts := []*PostFrontmatter{}
	xmlItems := xmlquery.Find(channel, "//entry")
	for _, entry := range xmlItems {
		entries, ok := c.OnXML_AtomEntry(r, entry, feed_language)
		if ok {
			posts = append(posts, entries...)
		}
	}

	slices.SortFunc(posts, func(a, b *PostFrontmatter) int {
		// Reverse chronological
		return -1 * cmpDateStr(a.Date, b.Date)
	})

	for i, post := range posts {
		if i < c.Config.MaxPostsPerFeed {
			c.SavePost(post)
		}
	}
}

func (c *Crawler) OnXML_AtomEntry(r *colly.Request, entry *xmlquery.Node, feed_language string) ([]*PostFrontmatter, bool) {
	feed_url := r.URL.String()

	post_id := xmlText(entry, "id")
	links := collectLinkHrefs(r, "link[@rel='alternate']", entry)
	if len(links) == 0 {
		// If we can't find a rel=alt link, fall back to all links
		links = collectLinkHrefs(r, "link", entry)
	}

	title := xmlText(entry, "title")

	// Whatever date we can find
	dateStr := xmlText(entry, "updated")
	if dateStr == "" {
		dateStr = xmlText(entry, "published")
	}
	date := fmtDate(dateStr)

	content := xmlText(entry, "content")
	categories := xmlPathAttrMultiple(entry, "category", "term")

	// Prefer languages set on the element itself
	language := xmlAttr(entry, "xml:lang")

	// Check a couple other places:
	if len(language) == 0 {
		language = xmlPathAttrSingle(entry, "content", "xml:lang")
	}
	if len(language) == 0 {
		language = xmlPathAttrSingle(entry, "summary", "xml:lang")
	}
	if len(language) == 0 {
		language = xmlPathAttrSingle(entry, "title", "xml:lang")
	}

	// Fall back to the feed language (if any)
	if len(language) == 0 {
		language = feed_language
	}

	// TODO, parse type: https://validator.w3.org/feed/docs/atom.html#text
	//       if type=html, convert back to plain text
	description := xmlText(entry, "summary")

	if title == "" {
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

	found := []*PostFrontmatter{}
	for _, link := range links {
		if strings.HasPrefix(link, "/") {
			// TODO: add support for xml:base, it's standard enough that it should be supported
			continue
		}
		if !isWebLink(link) {
			// This isn't a web link
			continue
		}

		post := NewPostFrontmatter(feed_url, post_id, link)
		post.WithTitle(title)
		post.WithDescription(description)
		post.WithDate(date)
		post.WithContent(content)
		post.WithFeedLink(feed_url)
		post.WithCategories(categories)
		post.WithLanguage(language)

		if isBlockedPost(link, title, post.Params.Id, c.Config) {
			continue
		}
		if blocked, domain := isBlockedDomain(link, c.Config); blocked {
			log.Printf("Domain is blocked: %s", domain)
			continue
		}

		found = append(found, post)
	}
	return found, true
}
