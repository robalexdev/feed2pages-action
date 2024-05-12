package main

func main() {
	config := parseConfig()
	cleanupContentOutputDirs(config)
	crawler := NewCrawler(config)
	crawler.Crawl(config.FeedUrl)
}
