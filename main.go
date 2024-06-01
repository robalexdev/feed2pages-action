package main

func main() {
	config := parseConfig()
	cleanupContentOutputDirs(config)
	crawler := NewCrawler(config)
	crawler.Crawl(config.FeedUrls...)
	crawler.PurgeNoIndex()
}
