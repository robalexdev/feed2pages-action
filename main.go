package main

import (
	"errors"
	"fmt"
	"github.com/mmcdole/gofeed"
	"slices"
	"time"
)

func filterPost(item *MultiTypeItem, config Config) error {
	// Missing required fields
	if item.item.PublishedParsed == nil {
		return errMissingField("PublishedParsed")
	}
	if item.item.Link == "" {
		return errMissingField("Link")
	}
	if item.item.Title == "" {
		return errMissingField("Title")
	}

	// Filter out banned words
	if has, which := containsAny(item.item.Description, config.BlockWords...); has {
		return errBlockWord("Description", which)
	}
	if has, which := containsAny(item.item.Title, config.BlockWords...); has {
		return errBlockWord("Title", which)
	}
	if has, which := containsAny(item.item.Content, config.BlockWords...); has {
		return errBlockWord("Content", which)
	}

	// Blocked posts by title, GUID, or link
	if slices.Contains(config.BlockPosts, item.item.GUID) {
		return errBlockPost("GUID", item.item.Title)
	}
	if slices.Contains(config.BlockPosts, item.item.Title) {
		return errBlockPost("Title", item.item.Title)
	}
	if slices.Contains(config.BlockPosts, item.item.Link) {
		return errBlockPost("Link", item.item.Title)
	}

	// Blocked domains
	if isBlocked, which := isBlockedDomain(item.item.Link, config); isBlocked {
		return errors.New(fmt.Sprintf("Domain is blocked: %s", which))
	}
	return nil
}

func processPost(item *MultiTypeItem, feed *gofeed.Feed, config Config) (PostFrontmatter, error) {
	out := PostFrontmatter{}
	out.Params.Feed = *feed
	out.Params.Feed.Items = []*gofeed.Item{} // exclude the others posts
	out.Params.Post = *item.item

	postDate := unixEpoc()
	if item.item.PublishedParsed != nil {
		postDate = *item.item.PublishedParsed
	}
	out.Date = postDate.Format(time.RFC3339)
	age := time.Since(postDate)
	out.Params.PrettyAge = pretty(age)
	ageDays := int(age.Hours() / 24)

	// Filter out content that's too old
	if config.PostAgeLimitDays != nil && ageDays > *config.PostAgeLimitDays {
		return out, errors.New("Too old")
	}

	// The description is one of:
	//  - description from the feed
	//  - the content from the feed
	out.Params.Post.Description = firstNonEmpty(
		out.Params.Post.Description,
		out.Params.Post.Content,
	)

	err := filterPost(item, config)
	if err != nil {
		return out, err
	}

	// An RSS only field (not Atom or JSON feeds)
	if item.rss != nil {
		out.Params.CommentsLink = item.rss.Comments
	}

	// Reduce the size, we won't render it all anyway
	out.Title = truncateText(readable(item.item.Title), 200)
	out.Params.Post.Description = truncateText(out.Params.Post.Description, 1024)
	out.Params.Post.Content = truncateText(out.Params.Post.Content, 1024)

	return out, nil
}

func processFeed(feedId string, feedDetails FeedDetails, config Config) ([]PostFrontmatter, *FeedFrontmatter, []PendingDiscover) {
	fp := NewParser()
	parsedFeed, mergedItems, err := fp.ParseURLExtended(feedDetails.Link)
	if err != nil {
		fmt.Printf("Unable to parse feed: %v %v", feedDetails, err)
		return nil, nil, nil
	}

	following := FeedFrontmatter{}
	following.Title = parsedFeed.Title
	following.Description = parsedFeed.Description
	following.Params.Feed = *parsedFeed
	following.Params.Id = feedId

	if isEmpty(following.Title) {
		following.Title = feedDetails.Title
	}
	if isEmpty(following.Description) {
		following.Description = feedDetails.Text
	}
	following.Date = bestFeedDate(parsedFeed)

	// Store posts elsewhere
	following.Params.Feed.Items = []*gofeed.Item{}

	reading := []PostFrontmatter{}
	for _, post := range mergedItems {
		processed, err := processPost(&post, parsedFeed, config)
		processed.Params.FeedId = feedId
		if err != nil {
			fmt.Printf("  Excluding post: %v\n", err)
			continue
		}
		reading = append(reading, processed)
	}

	// Parse blogroll
	discoverUrls := processBlogroll(parsedFeed)
	return reading, &following, discoverUrls
}

func main() {
	config := parseConfig()

	readingOutputPath := contentPath(firstNonEmpty(config.ReadingFolderName, DEFAULT_READING_FOLDER))
	followingOutputPath := contentPath(firstNonEmpty(config.FollowingFolderName, DEFAULT_FOLLOWING_FOLDER))
	discoverOutputPath := contentPath(firstNonEmpty(config.DiscoverFolderName, DEFAULT_DISCOVER_FOLDER))
	mkdirIfNotExists(readingOutputPath)
	mkdirIfNotExists(followingOutputPath)
	mkdirIfNotExists(discoverOutputPath)
	rmGenerated(READING_PREFIX, readingOutputPath)
	rmGenerated(FOLLOWING_PREFIX, followingOutputPath)
	rmGenerated(DISCOVER_PREFIX, discoverOutputPath)

	allReading := []PostFrontmatter{}
	allFollowing := []FeedFrontmatter{}
	allBlogrollURLs := []PendingDiscover{}
	for id, feedDetails := range config.Feeds {
		fmt.Printf("Processing feed: %v\n", feedDetails)
		feedId := fmt.Sprintf("%d", id)
		reading, following, discover := processFeed(feedId, feedDetails, config)
		if following != nil {
			allFollowing = append(allFollowing, *following)
		}
		reading = sortAndLimitPosts(reading, config.MaxPostsPerFeed)
		fmt.Printf("  got %d more items for reading list\n", len(reading))
		allReading = append(allReading, reading...)
		fmt.Printf("  discovered %d more feeds\n", len(discover))
		allBlogrollURLs = append(allBlogrollURLs, discover...)
	}
	allReading = sortAndLimitPosts(allReading, config.MaxPosts)
	fmt.Printf("Items in reading list: %d\n", len(allReading))
	for _, following := range allFollowing {
		path := generatedFilePath(followingOutputPath, FOLLOWING_PREFIX, following.Params.Id)
		writeYaml(following, path)
	}
	for _, reading := range allReading {
		path := generatedFilePath(readingOutputPath, READING_PREFIX, safeGUID(reading))
		writeYaml(reading, path)
	}

	allBlogrolls := discoverMoreFeeds(allBlogrollURLs, config)
	for id, discover := range allBlogrolls {
		path := generatedFilePath(discoverOutputPath, DISCOVER_PREFIX, fmt.Sprintf("%d", id))
		writeYaml(discover, path)
	}
}
