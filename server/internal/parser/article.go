package parser

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ParseArticleHTML extracts the readable article body text from an AP article page.
//
// It joins paragraph text with double newlines. If no recognizable article body is
// found, it returns an empty content string and no error (photo essays, live updates,
// and video-only pages are intentionally left empty).
func ParseArticleHTML(html []byte) (content string, err error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return "", fmt.Errorf("parse html: %w", err)
	}

	body := doc.Find("article, div.Article, div.article-body, div.RichTextStoryBody, div.RichTextArticleBody").First()
	if body.Length() == 0 {
		return "", nil
	}

	var paragraphs []string
	body.Find("p").Each(func(_ int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" {
			paragraphs = append(paragraphs, text)
		}
	})

	if len(paragraphs) == 0 {
		return "", nil
	}

	return strings.Join(paragraphs, "\n\n"), nil
}
