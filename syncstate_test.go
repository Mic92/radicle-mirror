package main

import (
	"testing"
	"time"
)

// a sync covers everything present at fetch time, so it must collapse
// all older queued webhook events instead of re-syncing per event
func TestSyncedAtCollapsesEventBacklog(t *testing.T) {
	s := newSyncState()
	fetchStart := time.Unix(10000, 0)
	event := time.Unix(9000, 0)

	s.markSynced(1, syncedAt(event, fetchStart))

	if !s.upToDate(1, fetchStart.Add(-clockSkewMargin)) {
		t.Error("events pushed before the fetch should be covered by the sync")
	}
	if s.upToDate(1, fetchStart) {
		t.Error("events near fetch start must re-sync (clock skew safety)")
	}
}

func TestSyncedAtKeepsNewerEvent(t *testing.T) {
	fetchStart := time.Unix(10000, 0)
	event := fetchStart.Add(time.Minute)
	if got := syncedAt(event, fetchStart); !got.Equal(event) {
		t.Errorf("got %v, want %v", got, event)
	}
}

func TestSyncStateUpToDate(t *testing.T) {
	s := newSyncState()
	t0 := time.Unix(1000, 0)
	t1 := time.Unix(2000, 0)

	// unknown repo is never up to date
	if s.upToDate(1, t0) {
		t.Error("unknown repo reported up to date")
	}

	s.markSynced(1, t1)
	if !s.upToDate(1, t1) {
		t.Error("same timestamp should be up to date")
	}
	if !s.upToDate(1, t0) {
		t.Error("older push should be up to date")
	}
	if s.upToDate(1, t1.Add(time.Second)) {
		t.Error("newer push should not be up to date")
	}
}
