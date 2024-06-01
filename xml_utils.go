package main

import (
	"github.com/antchfx/xmlquery"
	"github.com/gocolly/colly/v2"
	"strings"
)

func xmlText(node *xmlquery.Node, xpathStr string) string {
	found := xmlquery.FindOne(node, xpathStr)
	if found == nil {
		return ""
	}
	return strings.TrimSpace(found.InnerText())
}

func xmlTextMultiple(node *xmlquery.Node, xpathStr string) []string {
	found := xmlquery.Find(node, xpathStr)
	if len(found) < 1 {
		return []string{}
	}
	res := []string{}
	for _, node := range found {
		res = append(res, node.InnerText())
	}
	return res
}

func xmlAttr(node *xmlquery.Node, attrName string) string {
	return strings.TrimSpace(node.SelectAttr(attrName))
}

func xmlPathAttrMultiple(node *xmlquery.Node, xpathStr, attrName string) []string {
	found := xmlquery.Find(node, xpathStr)
	if len(found) < 1 {
		return []string{}
	}
	res := []string{}
	for _, node := range found {
		res = append(res, xmlAttr(node, attrName))
	}
	return res
}

func collectLinkHrefs(r *colly.Request, selectExpr string, node *xmlquery.Node) []string {
	links := xmlquery.Find(node, selectExpr)
	linkUrls := []string{}
	for _, link := range links {
		url := xmlAttr(link, "href")
		url = strings.TrimSpace(url)
		if url != "" {
			url = r.AbsoluteURL(url)
			linkUrls = append(linkUrls, url)
		}
	}
	return linkUrls
}
