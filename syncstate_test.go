package main

import (
	"testing"
	"time"
)

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
