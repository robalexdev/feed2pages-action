package main

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/go-yaml/yaml"
	"github.com/mmcdole/gofeed"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func anyContains(target string, searchAmong ...string) (bool, string) {
	for _, search := range searchAmong {
		if strings.Contains(search, target) {
			return true, search
		}
	}
	return false, ""
}

func containsAny(search string, targets ...string) (bool, string) {
	// Case insensitive
	search = strings.ToLower(search)
	for _, target := range targets {
		if strings.Contains(search, strings.ToLower(target)) {
			return true, target
		}
	}
	return false, ""
}

func mkdirIfNotExists(path string) {
	err := os.MkdirAll(path, 0755)
	if err != nil {
		panicf("Unable to create directory: %s: %e", path, err)
	}
}

func rmGenerated(prefix, path string) {
	files, err := os.ReadDir(path)
	if err != nil {
		panicf("Unable to read directory: %s: %e", path, err)
	}
	for _, f := range files {
		if strings.HasPrefix(f.Name(), prefix) {
			os.Remove(filepath.Join(path, f.Name()))
		}
	}
}

func pluralizeAgo(s string, i int) string {
	if i == 1 {
		return fmt.Sprintf("one %s ago", s)
	} else {
		return fmt.Sprintf("%d %ss ago", i, s)
	}
}

func pretty(duration time.Duration) string {
	HOURS_PER_DAY := 24
	HOURS_PER_MONTH := HOURS_PER_DAY * 30
	HOURS_PER_YEAR := HOURS_PER_MONTH * 12
	hoursAgo := int(duration.Hours())
	if hoursAgo > HOURS_PER_YEAR {
		t := hoursAgo / HOURS_PER_YEAR
		return pluralizeAgo("year", t)
	} else if hoursAgo > HOURS_PER_MONTH {
		t := hoursAgo / HOURS_PER_MONTH
		return pluralizeAgo("month", t)
	} else if hoursAgo > 2*HOURS_PER_DAY {
		t := hoursAgo / HOURS_PER_DAY
		return pluralizeAgo("day", t)
	} else if hoursAgo > HOURS_PER_DAY {
		return "yesterday"
	} else {
		return "today"
	}
}

func firstNonEmpty(options ...string) string {
	for _, option := range options {
		if !isEmpty(option) {
			return strings.TrimSpace(option)
		}
	}
	return ""
}

func isEmpty(s string) bool {
	return 0 == len(strings.TrimSpace(s))
}

func parseConfig() Config {
	content, closer, err := readFile("feeds.yaml")
	if err != nil {
		panicf("Unable to parse config: %e", err)
	}
	defer closer.Close()
	config := Config{}
	decoder := yaml.NewDecoder(content)
	err = decoder.Decode(&config)
	if err != nil {
		panicf("Config decode error: %e", err)
	}
	// Parse the OPML file (local file or remote resource)
	config.Feeds, err = parseOpml(config.FeedUrl)
	if err != nil {
		panicf("Unable to parse OPML: %e", err)
	}
	return config
}

func safeGUID(post PostFrontmatter) string {
	id := post.Params.Post.Link
	if post.Params.Post.GUID != "" {
		id = post.Params.Post.GUID
	}
	b := md5.Sum([]byte(id))
	s := hex.EncodeToString(b[:])
	return strings.Replace(s, "=", "", -1)
}

func sortAndLimitPosts(posts []PostFrontmatter, limit *int) []PostFrontmatter {
	sort.Slice(
		posts,
		func(i, j int) bool {
			return posts[i].Date > posts[j].Date
		},
	)
	if limit != nil {
		if *limit > len(posts) {
			return posts
		}
		return posts[:*limit]
	}
	return posts
}

func truncateText(s string, max int) string {
	if max > len(s) {
		return s
	}
	return s[:strings.LastIndexAny(s[:max], " .,:;-")]
}

func panicf(f string, a ...any) {
	panic(fmt.Sprintf(f, a...))
}

func errStrings(err error, s ...string) error {
	all := ""
	for _, next := range s {
		all = all + " " + next
	}
	return errors.New(fmt.Sprintf("%s: %e", all, err))
}

func errMissingField(field string) error {
	return errors.New(fmt.Sprintf("Missing required field: %s\n", field))
}

func errBlockWord(field string, word string) error {
	return errors.New(fmt.Sprintf("Skipping: %s content contains block word: %s", field, word))
}

func errBlockPost(field string, title string) error {
	return errors.New(fmt.Sprintf("Skipping: '%s': blocked by %s", title, field))
}

func unixEpoc() time.Time {
	return time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
}

func contentPath(folder string) string {
	return filepath.Join("content", folder)
}

func generatedFilePath(basePath, prefix, id string) string {
	name := fmt.Sprintf("%s%s.md", prefix, id)
	return filepath.Join(basePath, name)
}

func bestFeedDate(feed *gofeed.Feed) string {
	if feed.UpdatedParsed != nil {
		return feed.UpdatedParsed.Format(time.RFC3339)
	} else if feed.PublishedParsed != nil {
		return feed.PublishedParsed.Format(time.RFC3339)
	}
	return ""
}

// Check the RSS feed for source:blogroll entries
func processBlogrollBySourceBlogroll(feed *gofeed.Feed) []PendingDiscover {
	discoverUrls := []PendingDiscover{}
	// TODO: what if the namespace uses a different name?
	if sourceNS, has := feed.Extensions["source"]; has {
		if blogrolls, has := sourceNS["blogroll"]; has {
			for _, blogroll := range blogrolls {
				discovered := PendingDiscover{}
				discovered.RecommendedBy = feed.Link
				discovered.Link = blogroll.Value
				discoverUrls = append(discoverUrls, discovered)
			}
		}
	}
	return discoverUrls
}

func processBlogrollByWellKnownOpml(feed *gofeed.Feed) []PendingDiscover {
	if feed.Link == "" {
		return nil
	}

	url, err := url.Parse(feed.Link)
	if err != nil {
		fmt.Printf("URL parsing error: %e\n", err)
		return nil
	}

	domain := url.Scheme + "://" + url.Host
	wellKnown := domain + WELL_KNOWN_RECOMMENDATIONS_OPML
	if !httpHeadOk(wellKnown) {
		wellKnown = domain + WELL_KNOWN_RECOMMENDATIONS_JSON
		if httpHeadOk(wellKnown) {
			fmt.Printf("JSON recommendations not yet supported: %s\n", domain)
			return nil
		} else {
			// No well known file found
			return nil
		}
	}
	discovered := PendingDiscover{}
	discovered.RecommendedBy = feed.Link
	discovered.Link = wellKnown
	return []PendingDiscover{
		discovered,
	}
}

func processBlogrollByRelLink(feed *gofeed.Feed) []PendingDiscover {
	discoverUrls := []PendingDiscover{}
	if feed.Link == "" {
		return nil
	}

	body, err := httpGet(feed.Link)
	if err != nil {
		fmt.Printf("Unable to get feed link: %s: %e\n", feed.Link, err)
		return nil
	}
	defer body.Close()
	links, err := parseLinks(body)
	if err != nil {
		fmt.Printf("Unable to parse links in page: %s: %e\n", feed.Link, err)
		return nil
	}

	for _, link := range links {
		if link.Rel == "blogroll" {
			discovered := PendingDiscover{}
			discovered.RecommendedBy = feed.Link
			discovered.Link = link.Href
			if !strings.Contains(discovered.Link, "://") {
				// relative URL
				if strings.HasPrefix(discovered.Link, "/") {
					discovered.Link = feed.Link + discovered.Link
				} else {
					discovered.Link = feed.Link + "/" + discovered.Link
				}
			}
			discoverUrls = append(discoverUrls, discovered)
		}
	}
	return discoverUrls
}

func processBlogroll(feed *gofeed.Feed) []PendingDiscover {
	// Prefer blogroll specified in the RSS feed
	discoveredUrls := processBlogrollBySourceBlogroll(feed)
	if len(discoveredUrls) > 0 {
		return discoveredUrls
	}
	// Fall back to well known URLs
	//discoveredUrls = processBlogrollByWellKnownOpml(feed)
	//if len(discoveredUrls) > 0 {
	//	return discoveredUrls
	//}
	// Finally check link rel=blogroll
	return processBlogrollByRelLink(feed)
}

func sortDedupePendingDiscover(in []PendingDiscover) []PendingDiscover {
	sort.Slice(in, func(i, j int) bool {
		return in[i].Link < in[j].Link
	})
	last := PendingDiscover{}
	out := []PendingDiscover{}
	for _, check := range in {
		if check.Link != last.Link {
			out = append(out, check)
			last = check
		}
	}
	return out
}

func sortAndDedupeFeedDetails(in []FeedDetails) []FeedDetails {
	sort.Slice(in, func(i, j int) bool {
		return in[i].Link < in[j].Link
	})
	last := FeedDetails{}
	out := []FeedDetails{}
	for _, check := range in {
		if check.Link != last.Link {
			out = append(out, check)
			last = check
		}
	}
	return out
}

func parseFeedFromUrl(url string) (*gofeed.Feed, error) {
	fp := gofeed.NewParser()
	fp.RSSTranslator = NewCommentsTranslator()

	resp, err := httpGet(url)
	if err != nil {
		return nil, err
	}
	defer resp.Close()

	return fp.Parse(resp)
}

func discoverMoreFeeds(opmlUrls []PendingDiscover, config Config) []DiscoverFrontmatter {
	if config.DiscoverDepth == nil {
		fmt.Println("Skipping discovery, discover_depth is not set")
		return nil
	} else if *config.DiscoverDepth < 1 {
		fmt.Println("Skipping discovery, discover_depth is < 1")
		return nil
	}

	id := 0
	remainingDepth := *config.DiscoverDepth
	remainingRecs := 1000
	if config.MaxRecommendations != nil {
		remainingRecs = *config.MaxRecommendations
	}
	collected := map[string]DiscoverFrontmatter{}
	processed := map[string]bool{}
	for len(opmlUrls) > 0 && remainingDepth > 0 && remainingRecs > 0 {
		remainingDepth -= 1

		nextOpmlUrls := []PendingDiscover{}
		opmlUrls = sortDedupePendingDiscover(opmlUrls)

		for _, opmlInfo := range opmlUrls {
			remainingRecsForFeed := 100
			if config.MaxRecommendationsPerFeed != nil {
				remainingRecsForFeed = *config.MaxRecommendationsPerFeed
			}

			if _, has := processed[opmlInfo.Link]; has {
				// Already processed OPML URL
				continue
			} else {
				processed[opmlInfo.Link] = true
			}

			fmt.Printf("Discovering via: %v\n", opmlInfo)
			feedDetails, err := parseOpml(opmlInfo.Link)
			if err != nil {
				fmt.Printf("Unable to process OPML: %s: %e\n", opmlInfo.Link, err)
				continue
			}

			feedDetails = sortAndDedupeFeedDetails(feedDetails)

			for _, feedDetail := range feedDetails {
				fmt.Printf("Discovered: %s\n", feedDetail.Link)
				if found, has := collected[feedDetail.Link]; has {
					// Already processed link
					// Track a new recommender
					found.Params.RecommendedBy = append(found.Params.RecommendedBy, opmlInfo.RecommendedBy)
				} else {
					if isBlocked, _ := isBlockedDomain(feedDetail.Link, config); isBlocked {
						fmt.Printf("Blocked feed: %s\n", feedDetail.Link)
						continue
					}

					feed, err := parseFeedFromUrl(feedDetail.Link)
					if err != nil {
						fmt.Printf("Skipping feed: %s: %e\n", feedDetail.Link, err)
						// TODO add to collected somehow
						continue
					}

					newFeed := DiscoverFrontmatter{}
					newFeed.Params.Id = fmt.Sprintf("disc-%d", id)
					id += 1
					newFeed.Title = feed.Title
					newFeed.Description = feed.Description
					newFeed.Date = bestFeedDate(feed)
					newFeed.Params.Link = feed.Link
					if newFeed.Params.Link == "" {
						newFeed.Params.Link = feedDetail.Link
					}
					newFeed.Params.RecommendedBy = []string{opmlInfo.RecommendedBy}

					collected[feed.Link] = newFeed
					moreOpmlUrls := processBlogroll(feed)
					nextOpmlUrls = append(nextOpmlUrls, moreOpmlUrls...)

					remainingRecs -= 1
					if remainingRecs <= 0 {
						fmt.Println("Reached recommendation limit")
						break
					}
					remainingRecsForFeed -= 1
					if remainingRecsForFeed <= 0 {
						fmt.Println("Reached recommendation limit for feed")
						break
					}
				}
			}
		}
		// Process the next round of feeds
		opmlUrls = nextOpmlUrls
	}

	out := make([]DiscoverFrontmatter, 0, len(collected))
	for _, value := range collected {
		out = append(out, value)
	}
	return out
}

func isBlockedDomain(url string, config Config) (bool, string) {
	for _, blockedDomain := range config.BlockDomains {
		if isDomainOrSubdomain(url, blockedDomain) {
			return true, blockedDomain
		}
	}
	return false, ""
}
