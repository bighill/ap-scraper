package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestArticle_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	in := Article{
		URL:       "https://apnews.com/article/x",
		Title:     "Hello",
		ImageURL:  "https://example.com/i.jpg",
		Blurb:     "Short",
		PostedAt:  100,
		UpdatedAt: 200,
		ScrapedAt: 300,
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}

	var out Article
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if out != in {
		t.Fatalf("got %+v, want %+v", out, in)
	}
}

func TestArticle_JSONOmitempty(t *testing.T) {
	t.Parallel()

	in := Article{
		URL:       "https://apnews.com/article/y",
		Title:     "T",
		PostedAt:  1,
		UpdatedAt: 2,
		ScrapedAt: 3,
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if strings.Contains(s, "image_url") || strings.Contains(s, "blurb") {
		t.Fatalf("zero optional fields should be omitted: %s", s)
	}
}
