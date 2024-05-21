package main

import (
	"github.com/antchfx/xmlquery"
	"strings"
)

func xmlText(node *xmlquery.Node, xpathStr string) string {
	found := xmlquery.FindOne(node, xpathStr)
	if found == nil {
		return ""
	}
	return strings.TrimSpace(found.InnerText())
}

func xmlAttr(node *xmlquery.Node, attrName string) string {
	return strings.TrimSpace(node.SelectAttr(attrName))
}

func collectLinkHrefs(selectExpr string, node *xmlquery.Node) []string {
	links := xmlquery.Find(node, selectExpr)
	linkUrls := []string{}
	for _, link := range links {
		url := xmlAttr(link, "href")
		url = strings.TrimSpace(url)
		if url != "" {
			linkUrls = append(linkUrls, url)
		}
	}
	return linkUrls
}
