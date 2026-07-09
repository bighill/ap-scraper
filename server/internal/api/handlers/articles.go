package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"ap-scraper/internal/model"
)

// articleLister is the store surface needed for GET /articles (tests can pass a stub).
type articleLister interface {
	QueryAll(context.Context, bool) ([]model.Article, error)
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
func ListArticles(st articleLister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hidden := r.URL.Query().Get("hidden") == "1"
		items, err := st.QueryAll(r.Context(), hidden)
		if err != nil {
			log.Printf("api: query articles: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		if err := enc.Encode(items); err != nil {
			log.Printf("api: encode json: %v", err)
		}
	}
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
