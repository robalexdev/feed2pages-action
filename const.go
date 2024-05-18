package main

const USER_AGENT = "Feed2Pages/0.1"

const DEFAULT_READING_FOLDER = "reading"
const DEFAULT_FOLLOWING_FOLDER = "following"
const DEFAULT_DISCOVER_FOLDER = "discover"
const DEFAULT_NETWORK_FOLDER = "network"

const POST_PREFIX = "post-"
const FEED_PREFIX = "feed-"
const LINK_PREFIX = "link-"

const WELL_KNOWN_RECOMMENDATIONS_OPML = "/.well-known/recommendations.opml"

// This uses a float as a workaround for Go-Colly
// marshaling, which converts int to float
type NodeType = float64

const (
	NODE_TYPE_SEED NodeType = iota
	NODE_TYPE_FEED
	NODE_TYPE_WEBSITE
	NODE_TYPE_BLOGROLL
)
