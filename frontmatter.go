package main

// https://gohugo.io/content-management/front-matter/
type PostFrontmatter struct {
	Date        string     `yaml:"date"`
	Description string     `yaml:"description"`
	Title       string     `yaml:"title"`
	Params      PostParams `yaml:"params"`
}

type PostParams struct {
	Content string `yaml:"content"`
	FeedId  string `yaml:"feed_id"`
	Id      string `yaml:"id"`
	Link    string `yaml:"link"`
}

func NewPostFrontmatter(guid, link string) *PostFrontmatter {
	out := new(PostFrontmatter)
	out.Params.Id = buildSafeId(guid, link)
	out.Params.Link = link
	return out
}

func (f *PostFrontmatter) Save(config *ParsedConfig) {
	path := generatedFilePath(config.ReadingFolderName, POST_PREFIX, f.Params.Id)
	writeYaml(f, path)
}

func (f *PostFrontmatter) WithTitle(title string) {
	f.Title = truncateText(title, 200)
}

func (f *PostFrontmatter) WithDescription(description string) {
	f.Description = truncateText(readable(description), 200)
}

func (f *PostFrontmatter) WithDate(date string) {
	f.Date = date
}

func (f *PostFrontmatter) WithContent(content string) {
	f.Params.Content = truncateText(readable(content), 300)
}

func (f *PostFrontmatter) WithFeedLink(feed_link string) {
	f.Params.FeedId = buildSafeId("", feed_link)
}

type FeedFrontmatter struct {
	Date        string     `yaml:"date"`
	Description string     `yaml:"description"`
	Title       string     `yaml:"title"`
	Params      FeedParams `yaml:"params"`
}

type FeedParams struct {
	FeedLink string `yaml:"feedlink"`
	Id       string `yaml:"id"`
	Link     string `yaml:"link"`
}

func NewFeedFrontmatter(feed_url string) *FeedFrontmatter {
	out := new(FeedFrontmatter)
	out.Params.Id = buildSafeId("", feed_url)
	out.Params.FeedLink = feed_url
	return out
}

func (f *FeedFrontmatter) Save(isDirect bool, config *ParsedConfig) {
	var path string
	if isDirect {
		path = generatedFilePath(config.FollowingFolderName, FEED_PREFIX, f.Params.Id)
	} else {
		path = generatedFilePath(config.DiscoverFolderName, FEED_PREFIX, f.Params.Id)
	}
	writeYaml(f, path)
}

func (f *FeedFrontmatter) WithDate(date string) {
	f.Date = date
}

func (f *FeedFrontmatter) WithDescription(description string) {
	f.Description = truncateText(readable(description), 200)
}

func (f *FeedFrontmatter) WithTitle(title string) {
	f.Title = title
}

func (f *FeedFrontmatter) WithLink(link string) {
	f.Params.Link = link
}

type LinkFrontmatter struct {
	Params LinkParams `yaml:"params"`
}

type LinkParams struct {
	SourceURL      string `yaml:"source_url"`
	DestinationURL string `yaml:"destination_url"`
}

func NewLinkFrontmatter(source_url, destination_url string) *LinkFrontmatter {
	out := new(LinkFrontmatter)
	out.Params.SourceURL = source_url
	out.Params.DestinationURL = destination_url
	return out
}

func buildLinkId(source, dest string) string {
	return md5Hex(source + "\n" + dest)
}

func (f *LinkFrontmatter) Save(config *ParsedConfig) {
	id := buildLinkId(f.Params.SourceURL, f.Params.DestinationURL)
	path := generatedFilePath(config.NetworkFolderName, LINK_PREFIX, id)
	writeYaml(f, path)
}
