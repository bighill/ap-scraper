package parser

import (
	"strings"
	"testing"
)

const minimalArticlePage = `<!doctype html>
<html>
<body>
  <article>
    <p>First paragraph.</p>
    <p>  </p>
    <p>Second paragraph.</p>
  </article>
</body>
</html>`

func TestParseArticleHTML_minimal(t *testing.T) {
	t.Parallel()

	content, err := ParseArticleHTML([]byte(minimalArticlePage))
	if err != nil {
		t.Fatal(err)
	}
	want := "First paragraph.\n\nSecond paragraph."
	if content != want {
		t.Fatalf("got %q, want %q", content, want)
	}
}

func TestParseArticleHTML_emptyInput(t *testing.T) {
	t.Parallel()

	content, err := ParseArticleHTML(nil)
	if err != nil {
		t.Fatal(err)
	}
	if content != "" {
		t.Fatalf("want empty, got %q", content)
	}
}

func TestParseArticleHTML_noArticleBody(t *testing.T) {
	t.Parallel()

	html := `<html><body><div class="other"><p>Not an article.</p></div></body></html>`
	content, err := ParseArticleHTML([]byte(html))
	if err != nil {
		t.Fatal(err)
	}
	if content != "" {
		t.Fatalf("want empty, got %q", content)
	}
}

func TestParseArticleHTML_noParagraphs(t *testing.T) {
	t.Parallel()

	html := `<html><body><article><div>No paragraphs here.</div></article></body></html>`
	content, err := ParseArticleHTML([]byte(html))
	if err != nil {
		t.Fatal(err)
	}
	if content != "" {
		t.Fatalf("want empty, got %q", content)
	}
}

func TestParseArticleHTML_trimWhitespace(t *testing.T) {
	t.Parallel()

	html := `<html><body>
<div class="article-body">
  <p>  Leading and trailing spaces  </p>
  <p></p>
  <p>Another line.</p>
</div>
</body></html>`
	content, err := ParseArticleHTML([]byte(html))
	if err != nil {
		t.Fatal(err)
	}
	want := "Leading and trailing spaces\n\nAnother line."
	if content != want {
		t.Fatalf("got %q, want %q", content, want)
	}
}

func TestParseArticleHTML_returnsArticlePathPrefix(t *testing.T) {
	t.Parallel()

	html := strings.ReplaceAll(minimalArticlePage, "<article>", "<div class=\"Article\">")
	html = strings.ReplaceAll(html, "</article>", "</div>")
	content, err := ParseArticleHTML([]byte(html))
	if err != nil {
		t.Fatal(err)
	}
	if content == "" {
		t.Fatal("expected content from div.Article")
	}
}
