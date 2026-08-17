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
