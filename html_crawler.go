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
	metaSelAll := element.DOM.Find("meta[name='robots']")
	metaSelOurs := element.DOM.Find("meta[name='feed2pages/0.1']")

	// Prefer meta tags that are just for us
	metaSel := metaSelOurs
	if len(metaSel.Nodes) == 0 {
		metaSel = metaSelAll
	}

	isNofollow := false
	for i := range metaSel.Nodes {
		single := metaSel.Eq(i)
		content := single.AttrOr("content", "")
		if ContainsAnyString(content, META_ROBOT_NOINDEX_VARIANTS) {
			// Respect noindex
			log.Printf("NOINDEX: %s\n", page_url)
			c.db.TrackNoIndex(page_url)
		}
		if ContainsAnyString(content, META_ROBOT_NOFOLLOW_VARIANTS) {
			isNofollow = true
			log.Printf("NOFOLLOW: %s\n", page_url)
		}
	}

	// TODO, can I merge the QS as "link,a" ?
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
		if slices.Contains(rels, "canonical") {
			log.Printf("canonical URL: %s", href)
			c.Request(NODE_TYPE_WEBSITE, page_url, NODE_TYPE_CANONICAL, href, LINK_TYPE_LINK_REL_CANONICAL, r.Depth+1)
		}
	}

	// Always track rel=me, even when nofollow is set
	// This lets us verify rel=me links
	if slices.Contains(rels, "me") && slices.Contains(HTML_MIMES, t) {
		log.Printf("rel=me from HTML: %s", href)
		c.Request(NODE_TYPE_WEBSITE, page_url, NODE_TYPE_WEBSITE, href, LINK_TYPE_LINK_REL_ME, r.Depth+1)
	}
}
