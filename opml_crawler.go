package main

import (
	"log"

	"github.com/antchfx/xmlquery"
	"github.com/gocolly/colly/v2"
	"net/http"
)

func (c *Crawler) OnXML_Opml(_ *http.Header, r *colly.Request, opml *xmlquery.Node) {
	blogroll_url := r.URL.String()
	blogroll := NewBlogrollFrontmatter(blogroll_url)

	blogroll.WithTitle(xmlText(opml, "head/title"))
	blogroll.WithDescription(xmlText(opml, "head/description"))
	blogroll.WithDate(xmlText(opml, "head/dateModified"))

	xmlOutlines := xmlquery.Find(opml, "body//outline")
	for _, xmlOutline := range xmlOutlines {
		outline, ok := c.OnXML_OpmlOutline(r, xmlOutline)
		if ok {
			blogroll.Params.Outlines = append(blogroll.Params.Outlines, outline)
		}
	}

	if len(blogroll.Params.Outlines) > 0 {
		c.SaveBlogroll(blogroll)
	}
}

func (c *Crawler) OnXML_OpmlOutline(r *colly.Request, outline *xmlquery.Node) (BlogrollOutline, bool) {
	blogroll_url := r.URL.String()
	out := BlogrollOutline{}

	t := xmlAttr(outline, "type")
	text := xmlAttr(outline, "text")
	feedUrl := xmlAttr(outline, "xmlUrl")
	webUrl := xmlAttr(outline, "htmlUrl")
	url := xmlAttr(outline, "url")
	category := xmlAttr(outline, "category")

	// outline type
	// rss -> RSS feed
	// ?? -> Atom feed
	// include -> another outline to include inline
	if t == "include" {
		// TODO: add support when we have some hits here
		// the URL attribute is the one that should be present.
		log.Printf("OPML include: %s / %s / %s\n", feedUrl, webUrl, url)
		return out, false
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

	out.WithText(text)
	out.WithXmlUrl(feedUrl)
	out.WithHtmlUrl(webUrl)
	out.WithCategory(category)
	return out, true
}
