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
	QueryAll(context.Context) ([]model.Article, error)
}

// AllArticles returns JSON for every stored article (newest first by posted_at).
func AllArticles(st articleLister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := st.QueryAll(r.Context())
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
