package api

import "testing"

// TestNew_doesNotPanic verifies that registering the API routes and the
// web UI catch-all does not produce a gin route conflict at startup.
func TestNew_doesNotPanic(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("api.New panicked: %v", r)
		}
	}()

	_ = New(nil, ":0")
}
