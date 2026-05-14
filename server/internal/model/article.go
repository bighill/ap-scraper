package model

// Article holds normalized AP world-news metadata.
type Article struct {
	URL       string `json:"url"`
	Title     string `json:"title"`
	ImageURL  string `json:"image_url,omitempty"`
	Blurb     string `json:"blurb,omitempty"`
	PostedAt  int64  `json:"posted_at"`
	UpdatedAt int64  `json:"updated_at"`
	ScrapedAt int64  `json:"scraped_at"`
	IsHidden  bool   `json:"is_hidden"`
}
