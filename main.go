package main

import (
	"errors"
	"fmt"
	"github.com/mmcdole/gofeed"
	"slices"
	"time"
)

func filterPost(item *gofeed.Item, config Config) error {
	// Missing required fields
	if item.PublishedParsed == nil {
		return errMissingField("PublishedParsed")
	}
	if item.Link == "" {
		return errMissingField("Link")
	}
	if item.Title == "" {
		return errMissingField("Title")
	}

	// Filter out banned words
	if has, which := containsAny(item.Description, config.BlockWords...); has {
		return errBlockWord("Description", which)
	}
	if has, which := containsAny(item.Title, config.BlockWords...); has {
		return errBlockWord("Title", which)
	}
	if has, which := containsAny(item.Content, config.BlockWords...); has {
		return errBlockWord("Content", which)
	}

	// Blocked posts by title, GUID, or link
	if slices.Contains(config.BlockPosts, item.GUID) {
		return errBlockPost("GUID", item.Title)
	}
	if slices.Contains(config.BlockPosts, item.Title) {
		return errBlockPost("Title", item.Title)
	}
	if slices.Contains(config.BlockPosts, item.Link) {
		return errBlockPost("Link", item.Title)
	}

	// Blocked domains
	if isBlocked, which := isBlockedDomain(item.Link, config); isBlocked {
		return errors.New(fmt.Sprintf("Domain is blocked: %s", which))
	}
	return nil
}

func processPost(item *gofeed.Item, feed *gofeed.Feed, config Config) (PostFrontmatter, error) {
	out := PostFrontmatter{}
	out.Params.Feed = *feed
	out.Params.Feed.Items = []*gofeed.Item{} // exclude the others posts
	out.Params.Post = *item

	postDate := unixEpoc()
	if item.PublishedParsed != nil {
		postDate = *item.PublishedParsed
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
	if value, has := item.Custom[RSS_CUSTOM_COMMENT_KEY]; has {
		out.Params.CommentsLink = value
	}

	// Reduce the size, we won't render it all anyway
	out.Title = truncateText(readable(item.Title), 200)
	out.Params.Post.Description = truncateText(out.Params.Post.Description, 1024)
	out.Params.Post.Content = truncateText(out.Params.Post.Content, 1024)

	return out, nil
}

func processFeed(feedId string, feedDetails FeedDetails, config Config) ([]PostFrontmatter, *FeedFrontmatter, []PendingDiscover) {
	parsedFeed, err := parseFeedFromUrl(feedDetails.Link)
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
	for _, post := range parsedFeed.Items {
		processed, err := processPost(post, parsedFeed, config)
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
		fmt.Printf("  discovered %d more recommendation lists\n", len(discover))
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
