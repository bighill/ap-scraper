package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
)

// imageSettingGetter is the store surface needed for GET /settings/images.
type imageSettingGetter interface {
	ShowImages(context.Context) (bool, error)
}

// imageSettingSetter is the store surface needed for POST /settings/images.
type imageSettingSetter interface {
	SetShowImages(context.Context, bool) error
}

// GetShowImages returns the current app-level image visibility setting.
func GetShowImages(st imageSettingGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		show, err := st.ShowImages(r.Context())
		if err != nil {
			log.Printf("api: get show_images: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		if err := enc.Encode(map[string]bool{"show_images": show}); err != nil {
			log.Printf("api: encode json: %v", err)
		}
	}
}

// SetShowImages updates the app-level image visibility setting.
func SetShowImages(st imageSettingSetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Show *bool `json:"show_images"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if req.Show == nil {
			http.Error(w, "show_images required", http.StatusBadRequest)
			return
		}

		if err := st.SetShowImages(r.Context(), *req.Show); err != nil {
			log.Printf("api: set show_images: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
