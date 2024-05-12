package main

import (
	"bufio"
	readability "github.com/go-shiori/go-readability"
	"github.com/go-yaml/yaml"
	"io"
	"net/url"
	"os"
	"strings"
)

func readFile(path string) (io.Reader, io.Closer, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	return bufio.NewReader(f), f, nil
}

func writeYaml(o any, path string) {
	output, err := yaml.Marshal(o)
	if err != nil {
		panicf("YAML error: %e", err)
	}

	// Markdown uses `---` for YAML frontmatter
	sep := []byte("---\n")
	output = append(sep, output...)
	output = append(output, sep...)

	err = os.WriteFile(path, output, os.FileMode(int(0644)))
	if err != nil {
		panicf("Unable to write file: %s: %e", path, err)
	}
}

func readable(html string) string {
	fakeURL, err := url.Parse("")
	if err != nil {
		panic("Unable to build fake URL")
	}
	article, err := readability.FromReader(strings.NewReader(html), fakeURL)
	if err != nil {
		return ""
	}
	return article.TextContent
}

func isDomainOrSubdomain(questionURL string, domain string) bool {
	questionURL = strings.ToLower(questionURL)
	domain = strings.ToLower(domain)
	u, err := url.Parse(questionURL)
	if err != nil {
		return false
	}
	if u.Host == domain {
		return true
	}
	dotDomain := "." + domain
	if strings.HasSuffix(u.Host, dotDomain) {
		return true
	}
	return false
}
