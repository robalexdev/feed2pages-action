package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/go-yaml/yaml"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func setNoArchive(feed *FeedFrontmatter, headers *http.Header) {
	isNoarchive := false
	if headers != nil {
		for _, headerVal := range headers.Values("X-Robots-Tag") {
			if ContainsAnyString(headerVal, META_ROBOT_NOARCHIVE_VARIANTS) {
				isNoarchive = true
				break
			}
		}
	}
	feed.IsNoarchive(isNoarchive)
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

func parseConfig() *ParsedConfig {
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

	return config.Parse()
}

func truncateText(s string, max int) string {
	if max > len(s) {
		return s
	}
	idx := strings.LastIndexAny(s[:max], " .,:;-")
	if idx < 0 {
		idx = max
	}
	return s[:idx]
}

func panicf(f string, a ...any) {
	panic(fmt.Sprintf(f, a...))
}

func contentPath(folder string) string {
	return filepath.Join("content", folder)
}

func generatedFilePath(basePath, prefix, id string) string {
	name := fmt.Sprintf("%s%s.md", prefix, id)
	return filepath.Join(basePath, name)
}

func isBlockedDomain(url string, config *ParsedConfig) (bool, string) {
	for _, blockedDomain := range config.BlockDomains {
		if isDomainOrSubdomain(url, blockedDomain) {
			return true, blockedDomain
		}
	}
	return false, ""
}

func isBlockedPost(link, title, id string, config *ParsedConfig) bool {
	if _, has := config.BlockPosts[title]; has {
		log.Println("Blog blocked by title: %s", title)
		return true
	}
	if _, has := config.BlockPosts[link]; has {
		log.Println("Blog blocked by link: %s", link)
		return true
	}
	if _, has := config.BlockPosts[id]; has {
		log.Println("Blog blocked by ID: %s", id)
		return true
	}
	return false
}

func hasBlockWords(text string, config *ParsedConfig) (bool, string) {
	for _, blockedWord := range config.BlockWords {
		if strings.Contains(text, blockedWord) {
			return true, blockedWord
		}
	}
	return false, ""
}

func buildSafeId(id, link string) string {
	mustHash := false
	if len(id) < 8 {
		// Not unique enough, use link
		id = link
		mustHash = true
	}
	for _, c := range id {
		// ID isn't safe for the filesystem or URL path
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '-' {
			mustHash = true
			break
		}
	}
	if len(id) > 35 {
		// Hash to shorten
		mustHash = true
	}
	if mustHash {
		id = md5Hex(id)
	}
	return id
}

func buildSafePostId(feed_link, guid string) string {
	// Here we trust that the post ID is unique enough on a per-site basis
	return md5Hex(guid + " " + feed_link)
}

func md5Hex(s string) string {
	b := md5.Sum([]byte(s))
	return hex.EncodeToString(b[:])
}

func cleanupContentOutputDirs(config *ParsedConfig) {
	mkdirIfNotExists(config.ReadingFolderName)
	mkdirIfNotExists(config.FollowingFolderName)
	mkdirIfNotExists(config.DiscoverFolderName)
	mkdirIfNotExists(config.NetworkFolderName)
	if config.RemoveOldContent {
		rmGenerated(POST_PREFIX, config.ReadingFolderName)
		rmGenerated(FEED_PREFIX, config.FollowingFolderName)
		rmGenerated(FEED_PREFIX, config.DiscoverFolderName)
		rmGenerated(LINK_PREFIX, config.NetworkFolderName)
	}
}

func ContainsAnyString(search string, targets []string) bool {
	for _, target := range targets {
		if strings.Contains(search, target) {
			return true
		}
	}
	return false
}

func fmtDate(in string) string {
	t, err := ParseDate(in)
	if err != nil {
		return OLD_DATE_RFC3339
	}
	return t.Format(time.RFC3339)
}
