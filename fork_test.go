package main

import (
	"testing"

	"github.com/Mic92/radicle/github"
)

func TestSkipRepo(t *testing.T) {
	s := Server{mirroredForks: map[string]bool{"Mic92/nixpkgs": true}}
	cases := []struct {
		fullName string
		fork     bool
		want     bool
	}{
		{"Mic92/dotfiles", false, false}, // regular repo: mirrored
		{"Mic92/some-fork", true, true},  // fork: skipped
		{"Mic92/nixpkgs", true, false},   // allow-listed fork: mirrored
		{"Mic92/nixpkgs", false, false},  // allow-list entry that is no fork
	}
	for _, c := range cases {
		repo := &github.Repository{FullName: c.fullName, Fork: c.fork}
		if got := s.skipRepo(repo); got != c.want {
			t.Errorf("skipRepo(%q, fork=%v) = %v, want %v", c.fullName, c.fork, got, c.want)
		}
	}
}
