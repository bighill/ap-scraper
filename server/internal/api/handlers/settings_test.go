package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type stubImageGetter struct {
	show bool
	err  error
}

func (s stubImageGetter) ShowImages(ctx context.Context) (bool, error) {
	return s.show, s.err
}

type stubImageSetter struct {
	setTo *bool
	err   error
}

func (s *stubImageSetter) SetShowImages(ctx context.Context, show bool) error {
	s.setTo = &show
	return s.err
}

func TestGetShowImages_success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(GetShowImages(stubImageGetter{show: true}))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", ct)
	}

	var got map[string]bool
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got["show_images"] != true {
		t.Fatalf("got %+v", got)
	}
}

func TestGetShowImages_storeError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(GetShowImages(stubImageGetter{err: errors.New("boom")}))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestSetShowImages_successTrue(t *testing.T) {
	t.Parallel()

	s := &stubImageSetter{}
	srv := httptest.NewServer(SetShowImages(s))
	t.Cleanup(srv.Close)

	resp, err := http.Post(srv.URL, "application/json", strings.NewReader(`{"show_images":true}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if s.setTo == nil || *s.setTo != true {
		t.Fatalf("setTo = %v", s.setTo)
	}
}

func TestSetShowImages_successFalse(t *testing.T) {
	t.Parallel()

	s := &stubImageSetter{}
	srv := httptest.NewServer(SetShowImages(s))
	t.Cleanup(srv.Close)

	resp, err := http.Post(srv.URL, "application/json", strings.NewReader(`{"show_images":false}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if s.setTo == nil || *s.setTo != false {
		t.Fatalf("setTo = %v", s.setTo)
	}
}

func TestSetShowImages_missingField(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(SetShowImages(&stubImageSetter{}))
	t.Cleanup(srv.Close)

	resp, err := http.Post(srv.URL, "application/json", strings.NewReader(`{"other":true}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestSetShowImages_badJSON(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(SetShowImages(&stubImageSetter{}))
	t.Cleanup(srv.Close)

	resp, err := http.Post(srv.URL, "application/json", strings.NewReader("not json"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestSetShowImages_nonBoolean(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(SetShowImages(&stubImageSetter{}))
	t.Cleanup(srv.Close)

	resp, err := http.Post(srv.URL, "application/json", strings.NewReader(`{"show_images":"yes"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestSetShowImages_storeError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(SetShowImages(&stubImageSetter{err: errors.New("fail")}))
	t.Cleanup(srv.Close)

	resp, err := http.Post(srv.URL, "application/json", strings.NewReader(`{"show_images":true}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status %d", resp.StatusCode)
	}
}
