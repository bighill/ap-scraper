package model

// Article holds normalized AP world-news metadata plus optional full content.
type Article struct {
	ID               int64  `json:"id"`
	URL              string `json:"url"`
	Title            string `json:"title"`
	ImageURL         string `json:"image_url,omitempty"`
	Blurb            string `json:"blurb,omitempty"`
	Content          string `json:"content"`
	PostedAt         int64  `json:"posted_at"`
	UpdatedAt        int64  `json:"updated_at"`
	ScrapedAt        int64  `json:"scraped_at"`
	ContentScrapedAt int64  `json:"content_scraped_at"`
	IsHidden         bool   `json:"is_hidden"`
}
