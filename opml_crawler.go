package main

import (
	"github.com/antchfx/xmlquery"
	"github.com/gocolly/colly/v2"
)

func (c *Crawler) OnXML_OpmlOutline(r *colly.Request, outline *xmlquery.Node) {
	blogroll_url := r.URL.String()

	feedUrl := xmlAttr(outline, "xmlUrl")
	webUrl := xmlAttr(outline, "htmlUrl")

	if feedUrl != "" {
		feedUrl = r.AbsoluteURL(feedUrl)
		c.Request(NODE_TYPE_BLOGROLL, blogroll_url, NODE_TYPE_FEED, feedUrl, LINK_TYPE_FROM_OPML, r.Depth+1)
	}
	if webUrl != "" {
		webUrl = r.AbsoluteURL(webUrl)
		c.Request(NODE_TYPE_BLOGROLL, blogroll_url, NODE_TYPE_WEBSITE, webUrl, LINK_TYPE_FROM_OPML, r.Depth+1)
	}
}
