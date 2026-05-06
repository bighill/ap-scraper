# AP Scraper

Go CLI that scrapes AP world news stories from [apnews.com/world-news](https://apnews.com/world-news), normalizes metadata, stores it in local SQLite, and exposes JSON query commands.

## Current behavior

- Source page: `https://apnews.com/world-news`
- Parses `div.PagePromo` story cards and keeps article URLs matching `https://apnews.com/article/...`
- Captures per story:
  - `url` (primary key)
  - `title`
  - `image_url`
  - `blurb`
  - `posted_at` (ms epoch from `data-posted-date-timestamp`)
  - `updated_at` (ms epoch from `data-updated-date-timestamp`)
  - `scraped_at` (ms epoch set at runtime)
- Deduplicates by canonical URL within each scrape run before persistence
- Upserts by `url` into SQLite
- Applies retention every scrape: delete rows where `posted_at < now - 5 days` (UTC basis)

## CLI

- `go run ./cmd/apnews scrape`
  - Default behavior: fetch live HTML from AP and refresh cache file before parsing
- `go run ./cmd/apnews scrape --use-cache`
  - Parse cached HTML only
  - Errors clearly if cache file is missing
- `go run ./cmd/apnews query`
  - Returns recent rows as JSON
  - Default limit: 25
- `go run ./cmd/apnews query --limit N`
  - Returns up to `N` recent rows as JSON
- `go run ./cmd/apnews query all`
  - Returns all rows as JSON

## Paths and storage

- Database path: `data/apnews.db`
- Cache path: `data/world-news.cache.html`
- SQLite table: `world_news_stories`
  - `url` TEXT PRIMARY KEY
  - `title` TEXT NOT NULL
  - `image_url` TEXT
  - `blurb` TEXT
  - `posted_at` INTEGER NOT NULL
  - `updated_at` INTEGER NOT NULL
  - `scraped_at` INTEGER NOT NULL

## Helper scripts

- `bin/scrape.sh` - run default scrape (refresh cache)
- `bin/scrape-use-cache.sh` - run scrape with cached HTML
- `bin/query.sh` - query recent rows (default limit)
- `bin/query-all.sh` - query all rows
- `bin/build.sh` - build CLI binary
- `bin/test.sh` - run Go tests

## Constraints

- Respect [robots.txt](https://apnews.com/robots.txt), rate limits, and AP terms of use.
- Intended for personal or otherwise permitted use.
