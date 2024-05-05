package main

import (
	"golang.org/x/net/html"
	"io"
)

type Link struct {
	Rel  string
	Type string
	Href string
}

func processLinksFromDoc(node *html.Node, links []Link) []Link {
	if node.Type == html.ElementNode && node.Data == "link" {
		link := Link{}
		for _, attr := range node.Attr {
			if attr.Key == "href" {
				link.Href = attr.Val
			}
			if attr.Key == "type" {
				link.Type = attr.Val
			}
			if attr.Key == "rel" {
				link.Rel = attr.Val
			}
		}
		links = append(links, link)
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		links = processLinksFromDoc(child, links)
	}
	return links
}

func parseLinks(reader io.Reader) ([]Link, error) {
	doc, err := html.Parse(reader)
	if err != nil {
		return nil, err
	}
	links := []Link{}
	return processLinksFromDoc(doc, links), nil
}
