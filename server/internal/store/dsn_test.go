package store

import (
	"net/url"
	"strings"
	"testing"
)

func TestSQLiteDSN_pragmasAndFileScheme(t *testing.T) {
	t.Parallel()

	dsn, err := sqliteDSN("some-relative.db")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(dsn, "file:") {
		t.Fatalf("expected file: prefix, got %q", dsn)
	}
	if !strings.Contains(dsn, "?") {
		t.Fatalf("expected query string: %q", dsn)
	}

	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()["_pragma"]
	if len(q) < 2 {
		t.Fatalf("want 2 _pragma values, got %v", q)
	}
	var sawWal, sawBusy bool
	for _, p := range q {
		switch p {
		case "journal_mode(WAL)":
			sawWal = true
		case "busy_timeout(5000)":
			sawBusy = true
		}
	}
	if !sawWal || !sawBusy {
		t.Fatalf("pragmas: %v (wal=%v busy=%v)", q, sawWal, sawBusy)
	}
}

func TestSQLiteDSN_absolutePathNoDotDot(t *testing.T) {
	t.Parallel()

	dsn, err := sqliteDSN("ap.db")
	if err != nil {
		t.Fatal(err)
	}
	// Path portion should be absolute (no leading "..").
	if strings.Contains(dsn, "..") {
		t.Fatalf("dsn should not contain ..: %q", dsn)
	}
}
