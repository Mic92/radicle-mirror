package github

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTimestampUnmarshal(t *testing.T) {
	// webhook payloads encode pushed_at as a Unix epoch integer
	var webhook Repository
	if err := json.Unmarshal([]byte(`{"pushed_at":1782023169}`), &webhook); err != nil {
		t.Fatalf("cannot decode webhook timestamp: %v", err)
	}
	if got := webhook.PushedAt.Unix(); got != 1782023169 {
		t.Errorf("webhook: got %d, want 1782023169", got)
	}

	// REST API encodes pushed_at as an RFC3339 string
	var rest Repository
	if err := json.Unmarshal([]byte(`{"pushed_at":"2026-06-21T06:07:48Z"}`), &rest); err != nil {
		t.Fatalf("cannot decode rest timestamp: %v", err)
	}
	want := time.Date(2026, 6, 21, 6, 7, 48, 0, time.UTC)
	if !rest.PushedAt.Equal(want) {
		t.Errorf("rest: got %v, want %v", rest.PushedAt.Time, want)
	}

	// null must not error
	var empty Repository
	if err := json.Unmarshal([]byte(`{"pushed_at":null}`), &empty); err != nil {
		t.Fatalf("cannot decode null timestamp: %v", err)
	}
}
