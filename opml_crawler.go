package main

import (
	"github.com/antchfx/xmlquery"
	"github.com/gocolly/colly/v2"
)

func (c *Crawler) OnXML_OpmlBody(r *colly.Request, _ *xmlquery.Node) {
	// We need to save frontmatter here since we are guessing
	// some of the OPML URLs (.well-known path)
	// If we get here than the URL returned a 200 that looks like OPML
	c.SaveFrontmatter(r)
}

func (c *Crawler) OnXML_OpmlOutline(r *colly.Request, outline *xmlquery.Node) {
	blogroll_url := r.URL.String()

	feedUrl := xmlAttr(outline, "xmlUrl")
	webUrl := xmlAttr(outline, "htmlUrl")

	if feedUrl != "" {
		feedUrl = r.AbsoluteURL(feedUrl)
		c.Request(NODE_TYPE_BLOGROLL, blogroll_url, NODE_TYPE_FEED, feedUrl, r.Depth+1)
	} else if webUrl != "" {
		webUrl = r.AbsoluteURL(webUrl)
		c.Request(NODE_TYPE_BLOGROLL, blogroll_url, NODE_TYPE_WEBSITE, webUrl, r.Depth+1)
	}
}
