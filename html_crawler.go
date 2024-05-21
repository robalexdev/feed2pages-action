package main

import (
	"github.com/gocolly/colly/v2"
	"log"
)

// Example:
// <link rel="blogroll" type="text/xml" href="https://feedland.com/opml?screenname=davewiner&catname=blogroll">
func (c *Crawler) OnHTML_RelLink(element *colly.HTMLElement) {
	r := element.Request
	page_url := r.URL.String()
	rel := element.Attr("rel")
	t := element.Attr("type")
	href := element.Attr("href")
	if href == "" {
		return
	}
	href = r.AbsoluteURL(href)
	if rel == "blogroll" && (t == "" || t == "text/xml" || t == "application/atom+xml") {
		log.Printf("Blogroll from HTML: %s", href)
		c.Request(NODE_TYPE_WEBSITE, page_url, NODE_TYPE_BLOGROLL, href, r.Depth+1)
	}
}
