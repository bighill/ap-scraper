package config

import "time"

const (
	DBPath       = "./data/apnews.db"
	WorldNewsURL = "https://apnews.com/world-news"
	HTTPAddr     = ":9191"
	WebUIDir     = "../web"
)

// ScrapeInterval is how often the scheduler runs a live scrape (fetch + ingest).
const ScrapeInterval = 2 * time.Hour

// FetchTimeout bounds HTTP GET of the world-news page.
const FetchTimeout = 30 * time.Second

// ContentFetchTimeout bounds each individual article page fetch.
const ContentFetchTimeout = 20 * time.Second

// ContentFetchDelay is the polite pause between article page fetches.
const ContentFetchDelay = 2 * time.Second

// ArticleRetentionPeriod: articles with posted_at older than this (relative to now, UTC) are deleted after each scrape.
const ArticleRetentionPeriod = 2 * 24 * time.Hour
