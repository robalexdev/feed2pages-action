package main

import (
	"fmt"
	"github.com/mmcdole/gofeed"
	"github.com/mmcdole/gofeed/rss"
)

type CommentsTranslator struct {
	defaultTranslator *gofeed.DefaultRSSTranslator
}

func NewCommentsTranslator() *CommentsTranslator {
	t := &CommentsTranslator{}
	t.defaultTranslator = &gofeed.DefaultRSSTranslator{}
	return t
}

func (ct *CommentsTranslator) Translate(feed interface{}) (*gofeed.Feed, error) {
	rss, found := feed.(*rss.Feed)
	if !found {
		return nil, fmt.Errorf("Feed did not match expected type of *rss.Feed")
	}

	f, err := ct.defaultTranslator.Translate(rss)
	if err != nil {
		return nil, err
	}

	// Collect comments
	for _, item := range f.Items {
		for _, r := range rss.Items {
			if r.GUID != nil && item.GUID == r.GUID.Value {
				if item.Custom == nil {
					item.Custom = map[string]string{}
				}
				item.Custom[RSS_CUSTOM_COMMENT_KEY] = r.Comments
				break
			}
		}
	}

	return f, nil
}
