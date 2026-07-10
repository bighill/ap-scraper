package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ap-scraper/internal/model"

	"github.com/gin-gonic/gin"
)

type stubLister struct {
	items []model.Article
	err   error
}

func (s stubLister) QueryAll(ctx context.Context, hidden bool) ([]model.Article, error) {
	return s.items, s.err
}

type stubGetter struct {
	article model.Article
	err     error
}

func (s stubGetter) QueryOne(ctx context.Context, id int64) (model.Article, error) {
	return s.article, s.err
}

type stubHider struct {
	hideURL   string
	unhideURL string
	changed   bool
	err       error
}

func (s *stubHider) HideArticle(ctx context.Context, url string) (bool, error) {
	s.hideURL = url
	return s.changed, s.err
}

func (s *stubHider) UnhideArticle(ctx context.Context, url string) (bool, error) {
	s.unhideURL = url
	return s.changed, s.err
}

type stubCounter struct {
	total   int
	visible int
	hidden  int
	err     error
}

func (s stubCounter) CountArticles(ctx context.Context) (int, int, int, error) {
	return s.total, s.visible, s.hidden, s.err
}

func TestListArticles_success(t *testing.T) {
	t.Parallel()

	want := []model.Article{
		{ID: 1, URL: "https://apnews.com/article/a", Title: "A", PostedAt: 1, UpdatedAt: 2, ScrapedAt: 3},
		{ID: 2, URL: "https://apnews.com/article/b", Title: "B", PostedAt: 4, UpdatedAt: 5},
	}
	srv := httptest.NewServer(ListArticles(stubLister{items: want}))
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

	var got []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("len %d, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i]["id"] != float64(w.ID) || got[i]["url"] != w.URL || got[i]["title"] != w.Title {
			t.Fatalf("idx %d: got %+v want %+v", i, got[i], w)
		}
		if _, hasContent := got[i]["content"]; hasContent {
			t.Fatalf("idx %d: content should be omitted", i)
		}
	}
}

func TestListArticles_full(t *testing.T) {
	t.Parallel()

	items := []model.Article{
		{ID: 1, URL: "u", Title: "T", Content: "body", PostedAt: 1, UpdatedAt: 2, ScrapedAt: 3},
	}
	srv := httptest.NewServer(ListArticles(stubLister{items: items}))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "?full=1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var got []model.Article
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Content != "body" {
		t.Fatalf("got %+v", got)
	}
}

func TestListArticles_hiddenQueryParam(t *testing.T) {
	t.Parallel()

	lister := stubLister{items: []model.Article{{ID: 1, URL: "u", Title: "T", ScrapedAt: 1}}}
	srv := httptest.NewServer(ListArticles(lister))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "?hidden=1")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestListArticles_queryError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(ListArticles(stubLister{err: errors.New("boom")}))
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

func TestListArticles_emptySlice(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(ListArticles(stubLister{items: []model.Article{}}))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var got []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("got %v, want empty slice", got)
	}
}

func TestGetArticle_success(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	want := model.Article{
		ID:       7,
		URL:      "https://apnews.com/article/x",
		Title:    "T",
		Content:  "body",
		PostedAt: 1,
	}
	r := gin.New()
	r.GET("/articles/:id", GetArticle(stubGetter{article: want}))

	req := httptest.NewRequest(http.MethodGet, "/articles/7", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}

	var got model.Article
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestGetArticle_notFound(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/articles/:id", GetArticle(stubGetter{err: sql.ErrNoRows}))

	req := httptest.NewRequest(http.MethodGet, "/articles/99", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status %d", w.Code)
	}
}

func TestGetArticle_invalidID(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/articles/:id", GetArticle(stubGetter{}))

	req := httptest.NewRequest(http.MethodGet, "/articles/notanumber", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d", w.Code)
	}
}

func TestGetArticle_storeError(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/articles/:id", GetArticle(stubGetter{err: errors.New("boom")}))

	req := httptest.NewRequest(http.MethodGet, "/articles/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status %d", w.Code)
	}
}

func TestHideArticle_success(t *testing.T) {
	t.Parallel()

	h := &stubHider{changed: true}
	srv := httptest.NewServer(HideArticle(h))
	t.Cleanup(srv.Close)

	body := `{"url":"https://apnews.com/article/x"}`
	resp, err := http.Post(srv.URL, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if h.hideURL != "https://apnews.com/article/x" {
		t.Fatalf("hideURL = %q", h.hideURL)
	}
}

func TestHideArticle_missingURL(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(HideArticle(&stubHider{}))
	t.Cleanup(srv.Close)

	body := `{"url":""}`
	resp, err := http.Post(srv.URL, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestHideArticle_badJSON(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(HideArticle(&stubHider{}))
	t.Cleanup(srv.Close)

	resp, err := http.Post(srv.URL, "application/json", strings.NewReader("not json"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestHideArticle_storeError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(HideArticle(&stubHider{err: errors.New("fail")}))
	t.Cleanup(srv.Close)

	body := `{"url":"https://apnews.com/article/x"}`
	resp, err := http.Post(srv.URL, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestHideArticle_notFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(HideArticle(&stubHider{changed: false}))
	t.Cleanup(srv.Close)

	body := `{"url":"https://apnews.com/article/missing"}`
	resp, err := http.Post(srv.URL, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestUnhideArticle_success(t *testing.T) {
	t.Parallel()

	h := &stubHider{changed: true}
	srv := httptest.NewServer(UnhideArticle(h))
	t.Cleanup(srv.Close)

	body := `{"url":"https://apnews.com/article/x"}`
	resp, err := http.Post(srv.URL, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if h.unhideURL != "https://apnews.com/article/x" {
		t.Fatalf("unhideURL = %q", h.unhideURL)
	}
}

func TestUnhideArticle_missingURL(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(UnhideArticle(&stubHider{}))
	t.Cleanup(srv.Close)

	body := `{"url":""}`
	resp, err := http.Post(srv.URL, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestUnhideArticle_storeError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(UnhideArticle(&stubHider{err: errors.New("fail")}))
	t.Cleanup(srv.Close)

	body := `{"url":"https://apnews.com/article/x"}`
	resp, err := http.Post(srv.URL, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestUnhideArticle_notFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(UnhideArticle(&stubHider{changed: false}))
	t.Cleanup(srv.Close)

	body := `{"url":"https://apnews.com/article/missing"}`
	resp, err := http.Post(srv.URL, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestArticleCounts_success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(ArticleCounts(stubCounter{total: 10, visible: 7, hidden: 3}))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}

	var got map[string]int
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got["total"] != 10 || got["visible"] != 7 || got["hidden"] != 3 {
		t.Fatalf("got %+v", got)
	}
}

func TestArticleCounts_storeError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(ArticleCounts(stubCounter{err: errors.New("boom")}))
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
