package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestUntrackedRepos(t *testing.T) {
	dir := t.TempDir()
	for _, p := range []string{"100/1", "100/2", "200/3"} {
		if err := os.MkdirAll(filepath.Join(dir, p), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "100/2.rid"), []byte("rad:z123\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := untrackedRepos(dir, map[int]bool{1: true, 3: true})
	if err != nil {
		t.Fatal(err)
	}
	want := []untrackedRepo{{Path: filepath.Join(dir, "100/2"), Rid: "rad:z123"}}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestUntrackedReposMissingDir(t *testing.T) {
	got, err := untrackedRepos(filepath.Join(t.TempDir(), "nope"), nil)
	if err != nil || got != nil {
		t.Errorf("got %v, %v; want nil, nil", got, err)
	}
}
