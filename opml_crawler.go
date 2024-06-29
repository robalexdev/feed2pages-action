package main

import (
	"log"

	"github.com/antchfx/xmlquery"
	"github.com/gocolly/colly/v2"
	"net/http"
)

func (c *Crawler) OnXML_OpmlOutline(_ *http.Header, r *colly.Request, outline *xmlquery.Node) {
	blogroll_url := r.URL.String()

	t := xmlAttr(outline, "type")
	feedUrl := xmlAttr(outline, "xmlUrl")
	webUrl := xmlAttr(outline, "htmlUrl")
	url := xmlAttr(outline, "url")

	// outline type
	// rss -> RSS feed
	// ?? -> Atom feed
	// include -> another outline to include inline
	if t == "include" {
		// TODO: add support when we have some hits here
		// the URL attribute is the one that should be present.
		log.Printf("OPML include: %s / %s / %s\n", feedUrl, webUrl, url)
	} else {
		// Just load all other content regardless of type
		// If it parses as RSS or Atom, handle it as such
		if feedUrl != "" {
			feedUrl = r.AbsoluteURL(feedUrl)
			c.Request(NODE_TYPE_BLOGROLL, blogroll_url, NODE_TYPE_FEED, feedUrl, LINK_TYPE_FROM_OPML, r.Depth+1)
		}
		if webUrl != "" {
			webUrl = r.AbsoluteURL(webUrl)
			c.Request(NODE_TYPE_BLOGROLL, blogroll_url, NODE_TYPE_WEBSITE, webUrl, LINK_TYPE_FROM_OPML, r.Depth+1)
		}
	}
}
