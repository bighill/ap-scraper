package parser

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"

	"ap-scraper/internal/model"

	"github.com/PuerkitoBio/goquery"
)

const (
	apHost        = "apnews.com"
	apArticlePath = "/article/"
)

// ParseWorldNewsHTML extracts story metadata from the AP world-news page HTML.
//
// Returned stories are deduplicated by canonical URL within this parse result.
// ScrapedAt is intentionally left as 0 for the caller to set at runtime.
func ParseWorldNewsHTML(html []byte) ([]model.Story, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	items := make([]model.Story, 0)
	seen := make(map[string]struct{})

	doc.Find("div.PagePromo").Each(func(_ int, s *goquery.Selection) {
		postedAt, _ := parseEpochMillisAttr(s, "data-posted-date-timestamp")
		updatedAt, _ := parseEpochMillisAttr(s, "data-updated-date-timestamp")

		titleLink := s.Find("h3.PagePromo-title a.Link").First()
		href, _ := titleLink.Attr("href")
		canonURL, ok := canonicalAPArticleURL(href)
		if !ok {
			return
		}

		title := strings.TrimSpace(titleLink.Find(".PagePromoContentIcons-text").First().Text())
		if title == "" {
			title = strings.TrimSpace(titleLink.Text())
		}
		if title == "" || postedAt == 0 || updatedAt == 0 {
			return
		}

		if _, exists := seen[canonURL]; exists {
			return
		}
		seen[canonURL] = struct{}{}

		blurbLink := s.Find("div.PagePromo-description a.Link").First()
		blurb := strings.TrimSpace(blurbLink.Find(".PagePromoContentIcons-text").First().Text())
		if blurb == "" {
			blurb = strings.TrimSpace(blurbLink.Text())
		}

		imageURL := extractImageURL(s)

		items = append(items, model.Story{
			URL:       canonURL,
			Title:     title,
			ImageURL:  imageURL,
			Blurb:     blurb,
			PostedAt:  postedAt,
			UpdatedAt: updatedAt,
			ScrapedAt: 0,
		})
	})

	return items, nil
}

func canonicalAPArticleURL(href string) (string, bool) {
	href = strings.TrimSpace(href)
	if href == "" {
		return "", false
	}

	u, err := url.Parse(href)
	if err != nil {
		return "", false
	}

	// Allow relative URLs.
	if !u.IsAbs() {
		u.Scheme = "https"
		u.Host = apHost
	}

	if !strings.EqualFold(u.Host, apHost) {
		return "", false
	}
	if !strings.HasPrefix(u.Path, apArticlePath) {
		return "", false
	}

	// Canonicalize: scheme/host lowercased, strip query/fragment.
	u.Scheme = "https"
	u.Host = apHost
	u.RawQuery = ""
	u.Fragment = ""

	return u.String(), true
}

func parseEpochMillisAttr(s *goquery.Selection, attr string) (int64, bool) {
	v, ok := s.Attr(attr)
	if !ok {
		return 0, false
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, false
	}

	// AP uses ms epoch; ParseInt handles it directly.
	n, err := parseInt64(v)
	if err != nil {
		return 0, false
	}
	return n, true
}

func extractImageURL(s *goquery.Selection) string {
	img := s.Find("div.PagePromo-media img.Image").First()
	if img.Length() == 0 {
		return ""
	}

	if src, ok := img.Attr("src"); ok {
		src = strings.TrimSpace(src)
		if src != "" {
			return src
		}
	}

	if srcset, ok := img.Attr("srcset"); ok {
		// srcset is "url1 640w, url2 1024w, ..."; take first URL token.
		srcset = strings.TrimSpace(srcset)
		if srcset == "" {
			return ""
		}
		first := strings.Split(srcset, ",")[0]
		fields := strings.Fields(strings.TrimSpace(first))
		if len(fields) > 0 {
			return fields[0]
		}
	}

	return ""
}

func parseInt64(s string) (int64, error) {
	// Avoid pulling in strconv in many call sites while keeping this file cohesive.
	// (The compiler will inline this anyway.)
	var n int64
	var sign int64 = 1

	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	if s[0] == '-' {
		sign = -1
		s = s[1:]
		if s == "" {
			return 0, fmt.Errorf("invalid")
		}
	}

	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid digit")
		}
		n = n*10 + int64(c-'0')
	}
	return n * sign, nil
}

