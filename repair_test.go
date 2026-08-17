package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func gitInit(t *testing.T, dir string, bare bool) {
	t.Helper()
	args := []string{"init", "-q"}
	if bare {
		args = append(args, "--bare")
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
}

func TestIsGitRepo(t *testing.T) {
	repo := t.TempDir()
	gitInit(t, repo, true)
	if !isGitRepo(repo) {
		t.Error("bare repo not recognized as git repo")
	}
	// empty leftover directory from a failed clone
	if isGitRepo(t.TempDir()) {
		t.Error("empty directory recognized as git repo")
	}
	if isGitRepo(filepath.Join(t.TempDir(), "missing")) {
		t.Error("missing path recognized as git repo")
	}
}

func TestHasBranches(t *testing.T) {
	empty := t.TempDir()
	gitInit(t, empty, true)
	if hasBranches(empty) {
		t.Error("empty repo reported as having branches")
	}

	work := t.TempDir()
	gitInit(t, work, false)
	if err := os.WriteFile(filepath.Join(work, "f"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"add", "f"},
		{"-c", "user.email=t@t", "-c", "user.name=t", "commit", "-qm", "c"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = work
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	if !hasBranches(work) {
		t.Error("repo with a commit reported as having no branches")
	}
}

func TestRidFromExistsError(t *testing.T) {
	out := "✗ Error: repository: git: attempt to reinitialize " +
		"'/var/lib/private/radicle-mirror/rad/storage/z3StaJpzQNGhhkPfhCtRxSmNZZpu9'; " +
		"class=Repository (6); code=Exists (-4)"
	if got := ridFromExistsError(out); got != "rad:z3StaJpzQNGhhkPfhCtRxSmNZZpu9" {
		t.Errorf("unexpected rid: %q", got)
	}
	if got := ridFromExistsError("some other error"); got != "" {
		t.Errorf("expected empty rid, got %q", got)
	}
}
