// Package extractor implements the skim and deep content extraction tiers
// (Change 6). WS1 delivers URL/HTML extraction; WS2/WS3 will add PDF and image.
package extractor

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/net/html"
)

const (
	skimMaxBodyBytes = 2 << 20 // 2 MiB — cap network read for skim pass
	skimMaxTextRunes = 2000    // characters of body text to retain
)

// Page type constants returned in URLExtractResult.PageType.
const (
	PageTypeUnknown = "unknown"
	PageTypeArticle = "article"
	PageTypeEvent   = "event"
	PageTypeLanding = "landing"
)

// URLExtractResult holds the data produced by a skim extraction pass on a URL.
type URLExtractResult struct {
	Title       string
	Description string
	MainText    string
	PageType    string
}

// URLExtractor fetches and parses web pages for the skim tier.
// It is safe for concurrent use.
type URLExtractor struct {
	client *http.Client
}

// NewURLExtractor returns a URLExtractor configured for the skim tier:
// 8 s total timeout, max 3 redirects, 2 MiB body cap.
func NewURLExtractor() *URLExtractor {
	return &URLExtractor{
		client: &http.Client{
			Timeout: 8 * time.Second,
			CheckRedirect: func(_ *http.Request, via []*http.Request) error {
				if len(via) >= 3 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
	}
}

// Extract fetches rawURL and returns lightweight metadata.
// A non-nil error is non-fatal for callers — the resource was already saved
// and the caller should simply leave it unchanged.
func (e *URLExtractor) Extract(ctx context.Context, rawURL string) (URLExtractResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return URLExtractResult{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "SelfSystems/1.0 (content-indexer)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := e.client.Do(req)
	if err != nil {
		return URLExtractResult{}, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return URLExtractResult{}, fmt.Errorf("http %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		return URLExtractResult{}, fmt.Errorf("unsupported content-type: %s", ct)
	}

	doc, err := html.Parse(io.LimitReader(resp.Body, skimMaxBodyBytes))
	if err != nil {
		return URLExtractResult{}, fmt.Errorf("parse html: %w", err)
	}

	result := extractFromDoc(doc)
	result.PageType = detectPageType(result)
	return result, nil
}

// ---- HTML parsing -----------------------------------------------------------

type parsedMeta struct {
	title       string
	description string
	ogTitle     string
	ogDesc      string
	textParts   []string
}

// skipTags are skipped entirely — their subtrees contribute no text.
var skipTags = map[string]bool{
	"script": true, "style": true, "nav": true, "footer": true,
	"aside": true, "noscript": true, "iframe": true, "svg": true,
}

func extractFromDoc(doc *html.Node) URLExtractResult {
	m := &parsedMeta{}
	walkNode(m, doc)

	title := firstNonEmpty(m.ogTitle, m.title)
	desc := firstNonEmpty(m.ogDesc, m.description)
	mainText := buildMainText(m.textParts)

	return URLExtractResult{
		Title:       strings.TrimSpace(title),
		Description: strings.TrimSpace(desc),
		MainText:    mainText,
	}
}

func walkNode(m *parsedMeta, n *html.Node) {
	if n.Type == html.ElementNode {
		tag := strings.ToLower(n.Data)

		if skipTags[tag] {
			return // skip entire subtree
		}

		switch tag {
		case "title":
			if m.title == "" {
				m.title = collectText(n)
			}
			return // don't re-walk children
		case "meta":
			extractMeta(m, n)
			return // meta is self-closing
		}
	}

	if n.Type == html.TextNode {
		text := strings.TrimSpace(n.Data)
		if len(text) > 15 {
			m.textParts = append(m.textParts, text)
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkNode(m, c)
	}
}

func extractMeta(m *parsedMeta, n *html.Node) {
	name := attrVal(n, "name")
	prop := attrVal(n, "property")
	content := attrVal(n, "content")

	switch {
	case strings.EqualFold(name, "description") && m.description == "":
		m.description = content
	case strings.EqualFold(prop, "og:title") && m.ogTitle == "":
		m.ogTitle = content
	case strings.EqualFold(prop, "og:description") && m.ogDesc == "":
		m.ogDesc = content
	}
}

func collectText(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return sb.String()
}

func buildMainText(parts []string) string {
	combined := strings.Join(parts, " ")
	if utf8.RuneCountInString(combined) > skimMaxTextRunes {
		runes := []rune(combined)
		combined = string(runes[:skimMaxTextRunes])
	}
	return combined
}

func attrVal(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if strings.EqualFold(attr.Key, key) {
			return attr.Val
		}
	}
	return ""
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

// ---- Page type detection ----------------------------------------------------

var eventKeywords = []string{
	"hackathon", "deadline", "apply by", "register by", "applications close",
	"conference", "workshop", "competition", "sign up by", "submit by",
	"applications open", "internship", "apply now", "registration open",
}

var articleSignals = []string{
	"min read", "written by", "published by", "posted on", "author:",
}

func detectPageType(r URLExtractResult) string {
	combined := strings.ToLower(r.Title + " " + r.Description + " " + truncate(r.MainText, 500))

	for _, kw := range eventKeywords {
		if strings.Contains(combined, kw) {
			return PageTypeEvent
		}
	}
	for _, kw := range articleSignals {
		if strings.Contains(combined, kw) {
			return PageTypeArticle
		}
	}
	return PageTypeUnknown
}

func truncate(s string, maxRunes int) string {
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	return string([]rune(s)[:maxRunes])
}
