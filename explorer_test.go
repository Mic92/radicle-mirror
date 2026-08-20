package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Mic92/radicle/github"
)

func TestExplorerLink(t *testing.T) {
	dir := t.TempDir()
	repo := &github.Repository{}
	repo.Id = 7
	repo.Owner.Id = 3
	s := &Server{
		reposPath:   dir,
		explorerURL: "https://app.radicle.xyz/nodes/seed.radicle.garden/{rid}/commits/{sha}",
	}

	if got := s.explorerLink(repo, "abc123"); got != "" {
		t.Errorf("expected empty link, got %q", got)
	}

	ridPath := filepath.Join(dir, "3", "7.rid")
	if err := os.MkdirAll(filepath.Dir(ridPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ridPath, []byte("rad:zDeadBeef\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	want := "https://app.radicle.xyz/nodes/seed.radicle.garden/rad:zDeadBeef/commits/abc123"
	if got := s.explorerLink(repo, "abc123"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	s.explorerURL = ""
	if got := s.explorerLink(repo, "abc123"); got != "" {
		t.Errorf("expected empty link with no template, got %q", got)
	}
}
