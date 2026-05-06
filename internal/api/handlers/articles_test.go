package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"ap-scraper/internal/model"
)

type stubLister struct {
	items []model.Article
	err   error
}

func (s stubLister) QueryAll(ctx context.Context) ([]model.Article, error) {
	return s.items, s.err
}

func TestAllArticles_success(t *testing.T) {
	t.Parallel()

	want := []model.Article{
		{URL: "https://apnews.com/article/a", Title: "A", PostedAt: 1, UpdatedAt: 2, ScrapedAt: 3},
		{URL: "https://apnews.com/article/b", Title: "B", PostedAt: 4, UpdatedAt: 5},
	}
	srv := httptest.NewServer(AllArticles(stubLister{items: want}))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", ct)
	}

	var got []model.Article
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("len %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].URL != want[i].URL || got[i].Title != want[i].Title {
			t.Fatalf("idx %d: got %+v want %+v", i, got[i], want[i])
		}
	}
}

func TestAllArticles_queryError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(AllArticles(stubLister{err: errors.New("boom")}))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestAllArticles_emptySlice(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(AllArticles(stubLister{items: []model.Article{}}))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var got []model.Article
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("got %v, want empty slice", got)
	}
}
