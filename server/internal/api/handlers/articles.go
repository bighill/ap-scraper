package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"ap-scraper/internal/model"

	"github.com/gin-gonic/gin"
)

// articleLister is the store surface needed for GET /articles (tests can pass a stub).
type articleLister interface {
	QueryAll(context.Context, bool) ([]model.Article, error)
}

// articleGetter is the store surface needed for GET /articles/:id.
type articleGetter interface {
	QueryOne(context.Context, int64) (model.Article, error)
}

// articleHider is the store surface needed for POST /articles/hide and /articles/unhide.
type articleHider interface {
	HideArticle(context.Context, string) (bool, error)
	UnhideArticle(context.Context, string) (bool, error)
}

// articleCounter is the store surface needed for GET /articles/count.
type articleCounter interface {
	CountArticles(context.Context) (total, visible, hidden int, err error)
}

// ListArticles returns JSON for stored articles, filtered by hidden status.
// Pass ?full=1 to include the content field in each article.
func ListArticles(st articleLister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hidden := r.URL.Query().Get("hidden") == "1"
		full := r.URL.Query().Get("full") == "1"

		items, err := st.QueryAll(r.Context(), hidden)
		if err != nil {
			log.Printf("api: query articles: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")

		if full {
			if err := enc.Encode(items); err != nil {
				log.Printf("api: encode json: %v", err)
			}
			return
		}

		summaries := make([]articleSummary, len(items))
		for i, a := range items {
			summaries[i] = articleSummary{
				ID:        a.ID,
				URL:       a.URL,
				Title:     a.Title,
				ImageURL:  a.ImageURL,
				Blurb:     a.Blurb,
				PostedAt:  a.PostedAt,
				UpdatedAt: a.UpdatedAt,
				ScrapedAt: a.ScrapedAt,
				IsHidden:  a.IsHidden,
			}
		}
		if err := enc.Encode(summaries); err != nil {
			log.Printf("api: encode json: %v", err)
		}
	}
}

type articleSummary struct {
	ID        int64  `json:"id"`
	URL       string `json:"url"`
	Title     string `json:"title"`
	ImageURL  string `json:"image_url,omitempty"`
	Blurb     string `json:"blurb,omitempty"`
	PostedAt  int64  `json:"posted_at"`
	UpdatedAt int64  `json:"updated_at"`
	ScrapedAt int64  `json:"scraped_at"`
	IsHidden  bool   `json:"is_hidden"`
}

// GetArticle returns a single article by database id, including its content.
func GetArticle(st articleGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		idParam := c.Param("id")
		id, err := strconv.ParseInt(idParam, 10, 64)
		if err != nil {
			c.String(http.StatusBadRequest, "invalid article id")
			return
		}

		article, err := st.QueryOne(c.Request.Context(), id)
		if err != nil {
			if isNotFound(err) {
				c.Status(http.StatusNotFound)
				return
			}
			log.Printf("api: get article %d: %v", id, err)
			c.String(http.StatusInternalServerError, "internal error")
			return
		}

		c.Header("Content-Type", "application/json; charset=utf-8")
		enc := json.NewEncoder(c.Writer)
		enc.SetIndent("", "  ")
		if err := enc.Encode(article); err != nil {
			log.Printf("api: encode json: %v", err)
		}
	}
}

func isNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

// HideArticle marks an article as hidden.
func HideArticle(st articleHider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			URL string `json:"url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if req.URL == "" {
			http.Error(w, "url required", http.StatusBadRequest)
			return
		}
		changed, err := st.HideArticle(r.Context(), req.URL)
		if err != nil {
			log.Printf("api: hide article: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !changed {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// UnhideArticle marks an article as visible.
func UnhideArticle(st articleHider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			URL string `json:"url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if req.URL == "" {
			http.Error(w, "url required", http.StatusBadRequest)
			return
		}
		changed, err := st.UnhideArticle(r.Context(), req.URL)
		if err != nil {
			log.Printf("api: unhide article: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !changed {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// ArticleCounts returns total, visible, and hidden article counts.
func ArticleCounts(st articleCounter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		total, visible, hidden, err := st.CountArticles(r.Context())
		if err != nil {
			log.Printf("api: count articles: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		if err := enc.Encode(map[string]int{
			"total":   total,
			"visible": visible,
			"hidden":  hidden,
		}); err != nil {
			log.Printf("api: encode json: %v", err)
		}
	}
}
