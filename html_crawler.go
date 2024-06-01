package main

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"log"
	"slices"
	"strings"
)

func (c *Crawler) OnHTML(element *colly.HTMLElement) {
	r := element.Request
	page_url := r.URL.String()
	page_type := r.Ctx.GetAny("target_type")
	if page_type != NODE_TYPE_WEBSITE {
		// This isn't supposed to be a website
		// Maybe we're seeing an HTML error for RSS feed?
		return
	}

	// Check for meta robots for opt-outs
	metaSel := element.DOM.Find("meta[name='robots']")
	isNofollow := false
	for i := range metaSel.Nodes {
		single := metaSel.Eq(i)
		content := single.AttrOr("content", "")
		if ContainsAnyString(content, META_ROBOT_NOINDEX_VARIANTS) {
			// Respect noindex
			c.db.TrackNoIndex(page_url)
			return
		}
		if ContainsAnyString(content, META_ROBOT_NOFOLLOW_VARIANTS) {
			isNofollow = true
			log.Printf("NOFOLLOW: %s\n", page_url)
		}
	}

	linkSel := element.DOM.Find("link")
	for i := range linkSel.Nodes {
		single := linkSel.Eq(i)
		c.OnHTML_Link(single, element.Request, isNofollow)
	}

	aSel := element.DOM.Find("a")
	for i := range aSel.Nodes {
		single := aSel.Eq(i)
		c.OnHTML_Link(single, element.Request, isNofollow)
	}
}

// Example:
// <link rel="blogroll" type="text/xml" href="https://feedland.com/opml?screenname=davewiner&catname=blogroll">
func (c *Crawler) OnHTML_Link(element *goquery.Selection, r *colly.Request, isNofollow bool) {
	page_url := r.URL.String()
	page_type := r.Ctx.GetAny("target_type")
	if page_type != NODE_TYPE_WEBSITE {
		// This isn't supposed to be a website
		// Maybe we're seeing an HTML error for RSS feed?
		return
	}
	// rel can be space separated
	rels := strings.Fields(strings.ToLower(element.AttrOr("rel", "")))
	t := strings.ToLower(element.AttrOr("type", ""))
	href, exists := element.Attr("href")
	if !exists {
		return
	}

	// TODO: track canonical links
	href = r.AbsoluteURL(href)

	if !isNofollow {
		// Don't find these links, nofollow was set
		if slices.Contains(rels, "blogroll") && slices.Contains(OPML_MIMES, t) {
			log.Printf("Blogroll from HTML: %s", href)
			c.Request(NODE_TYPE_WEBSITE, page_url, NODE_TYPE_BLOGROLL, href, LINK_TYPE_LINK_REL_BLOGROLL, r.Depth+1)
		}
		if slices.Contains(rels, "alternate") && slices.Contains(FEED_MIMES, t) {
			log.Printf("Feed from HTML: %s", href)
			c.Request(NODE_TYPE_WEBSITE, page_url, NODE_TYPE_FEED, href, LINK_TYPE_LINK_REL_ALT, r.Depth+1)
		}
	}

	// Always track rel=me, even when nofollow is set
	// This lets us verify rel=me links
	// We handle noindex easlier in the process
	if slices.Contains(rels, "me") && slices.Contains(HTML_MIMES, t) {
		log.Printf("rel=me from HTML: %s", href)
		c.Request(NODE_TYPE_WEBSITE, page_url, NODE_TYPE_WEBSITE, href, LINK_TYPE_LINK_REL_ME, r.Depth+1)
	}
}
