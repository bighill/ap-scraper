package parser

import (
	"strings"
	"testing"
)

const minimalPagePromo = `<!doctype html><html><body>
<div class="PagePromo" data-posted-date-timestamp="1700000000000" data-updated-date-timestamp="1700000001000">
  <h3 class="PagePromo-title"><a class="Link" href="/article/example-slug">
    <span class="PagePromoContentIcons-text">Headline</span>
  </a></h3>
  <div class="PagePromo-description"><a class="Link">
    <span class="PagePromoContentIcons-text">Deck text</span>
  </a></div>
  <div class="PagePromo-media"><img class="Image" src="https://cdn.example/img.jpg" alt=""></div>
</div>
</body></html>`

func TestParseWorldNewsHTML_minimalCard(t *testing.T) {
	t.Parallel()

	items, err := ParseWorldNewsHTML([]byte(minimalPagePromo))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("len %d", len(items))
	}
	got := items[0]
	if got.URL != "https://apnews.com/article/example-slug" {
		t.Fatalf("URL %q", got.URL)
	}
	if got.Title != "Headline" {
		t.Fatalf("Title %q", got.Title)
	}
	if got.Blurb != "Deck text" {
		t.Fatalf("Blurb %q", got.Blurb)
	}
	if got.ImageURL != "https://cdn.example/img.jpg" {
		t.Fatalf("ImageURL %q", got.ImageURL)
	}
	if got.PostedAt != 1700000000000 || got.UpdatedAt != 1700000001000 {
		t.Fatalf("timestamps %+v", got)
	}
	if got.ScrapedAt != 0 {
		t.Fatalf("ScrapedAt %d", got.ScrapedAt)
	}
}

func TestParseWorldNewsHTML_emptyInput(t *testing.T) {
	t.Parallel()

	items, err := ParseWorldNewsHTML(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("got %d items", len(items))
	}

	items, err = ParseWorldNewsHTML([]byte{})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("got %d items", len(items))
	}
}

func TestParseWorldNewsHTML_noMatchingCards(t *testing.T) {
	t.Parallel()

	html := `<html><body><div class="other">no promo</div></body></html>`
	items, err := ParseWorldNewsHTML([]byte(html))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("want 0, got %d", len(items))
	}
}

func TestParseWorldNewsHTML_dedupesSameURL(t *testing.T) {
	t.Parallel()

	html := `<html><body>
<div class="PagePromo" data-posted-date-timestamp="1" data-updated-date-timestamp="2">
  <h3 class="PagePromo-title"><a class="Link" href="https://apnews.com/article/dup"><span class="PagePromoContentIcons-text">One</span></a></h3>
  <div class="PagePromo-description"><a class="Link"><span class="PagePromoContentIcons-text">B</span></a></div>
</div>
<div class="PagePromo" data-posted-date-timestamp="1" data-updated-date-timestamp="2">
  <h3 class="PagePromo-title"><a class="Link" href="https://apnews.com/article/dup"><span class="PagePromoContentIcons-text">Two</span></a></h3>
  <div class="PagePromo-description"><a class="Link"><span class="PagePromoContentIcons-text">B</span></a></div>
</div>
</body></html>`
	items, err := ParseWorldNewsHTML([]byte(html))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("want 1, got %d", len(items))
	}
	if items[0].Title != "One" {
		t.Fatalf("first win: %q", items[0].Title)
	}
}

func TestCanonicalAPArticleURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		href string
		ok   bool
		want string
	}{
		{"", false, ""},
		{"https://example.com/article/x", false, ""},
		{"https://apnews.com/world-news", false, ""},
		{"/article/foo", true, "https://apnews.com/article/foo"},
		{"https://apnews.com/article/bar?q=1#h", true, "https://apnews.com/article/bar"},
		{"HTTPS://APNEWS.COM/article/Baz", true, "https://apnews.com/article/Baz"},
	}
	for _, tt := range tests {
		got, ok := canonicalAPArticleURL(tt.href)
		if ok != tt.ok || got != tt.want {
			t.Fatalf("href %q: got (%q, %v), want (%q, %v)", tt.href, got, ok, tt.want, tt.ok)
		}
	}
}

func TestParseWorldNewsHTML_skipsIncompleteCard(t *testing.T) {
	t.Parallel()

	// Missing updated timestamp → skipped.
	html := `<html><body><div class="PagePromo" data-posted-date-timestamp="1">
  <h3 class="PagePromo-title"><a class="Link" href="/article/x"><span class="PagePromoContentIcons-text">T</span></a></h3>
  <div class="PagePromo-description"><a class="Link"><span class="PagePromoContentIcons-text">B</span></a></div>
</div></body></html>`
	items, err := ParseWorldNewsHTML([]byte(html))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("want 0, got %+v", items)
	}
}

// Guardrail: output URLs must stay article-shaped (regression if selectors change).
func TestParseWorldNewsHTML_minimalCard_articlePath(t *testing.T) {
	t.Parallel()

	items, err := ParseWorldNewsHTML([]byte(minimalPagePromo))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatal(len(items))
	}
	if !strings.HasPrefix(items[0].URL, "https://apnews.com/article/") {
		t.Fatalf("URL %q", items[0].URL)
	}
}
