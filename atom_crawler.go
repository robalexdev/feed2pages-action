package main

import (
	"cmp"
	"github.com/antchfx/xmlquery"
	"github.com/gocolly/colly/v2"
	"log"
	"slices"
)

func (c *Crawler) OnXML_AtomFeed(r *colly.Request, channel *xmlquery.Node) {
	feed_url := r.URL.String()
	links := collectLinkHrefs(r, "link[@rel='alternate']", channel)
	title := xmlText(channel, "title")
	description := xmlText(channel, "subtitle")
	date := xmlText(channel, "updated")

	feed := NewFeedFrontmatter(feed_url)
	feed.WithDate(date)
	feed.WithTitle(title)
	feed.WithFeedType("atom")
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
		c.SaveFeed(feed, isDirect)

		log.Printf("Searching for blogroll in: %s", link)
		c.Request(NODE_TYPE_FEED, feed_url, NODE_TYPE_WEBSITE, link, LINK_TYPE_LINK_REL_ALT, r.Depth+1)
	}

	// Atom feeds don't have a blogroll syntax yet
	// Add here when they do

	c.CollectAtomEntries(r, channel)
}

func (c *Crawler) CollectAtomEntries(r *colly.Request, channel *xmlquery.Node) {
	// TODO: change collect depth
	if r.Depth > 2 {
		return
	}
	if c.Config.MaxPostsPerFeed < 1 {
		return
	}

	posts := []*PostFrontmatter{}
	xmlItems := xmlquery.Find(channel, "//entry")
	for _, entry := range xmlItems {
		entries, ok := c.OnXML_AtomEntry(r, entry)
		if ok {
			posts = append(posts, entries...)
		}
	}

	slices.SortFunc(posts, func(a, b *PostFrontmatter) int {
		return cmp.Compare(a.Date, b.Date)
	})

	for i, post := range posts {
		if i < c.Config.MaxPostsPerFeed {
			c.SavePost(post)
		}
	}
}

func (c *Crawler) OnXML_AtomEntry(r *colly.Request, entry *xmlquery.Node) ([]*PostFrontmatter, bool) {
	feed_url := r.URL.String()

	post_id := xmlText(entry, "id")
	links := collectLinkHrefs(r, "link[@rel='alternate']", entry)
	if len(links) == 0 {
		// If we can't find a rel=alt link, fall back to all links
		links = collectLinkHrefs(r, "link", entry)
	}

	title := xmlText(entry, "title")
	date := xmlText(entry, "updated")
	content := xmlText(entry, "content")
	categories := xmlPathAttrMultiple(entry, "category", "term")

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
		post := NewPostFrontmatter(post_id, link)
		post.WithTitle(title)
		post.WithDescription(description)
		post.WithDate(date)
		post.WithContent(content)
		post.WithFeedLink(feed_url)
		post.WithCategories(categories)

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
