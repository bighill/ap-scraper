package config

import "time"

// Static configuration (no env vars for now; add later if needed).

const (
	DBPath       = "./data/apnews.db"
	WorldNewsURL = "https://apnews.com/world-news"
	HTTPAddr     = ":9191"
	WebUIDir     = "../web"
)

// ScrapeInterval is how often the scheduler runs a live scrape (fetch + ingest).
const ScrapeInterval = 77 * time.Minute

// FetchTimeout bounds HTTP GET of the world-news page.
const FetchTimeout = 30 * time.Second

// ArticleRetentionPeriod: articles with posted_at older than this (relative to now, UTC) are deleted after each scrape.
const ArticleRetentionPeriod = 2 * 24 * time.Hour
