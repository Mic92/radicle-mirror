package main

import (
	"context"
	"fmt"
	"github.com/Mic92/radicle/github"
	"log/slog"
	"time"
)

// Polls github repositories for new commits (in case webhook doesn't work)
func (s Server) refreshRepos() ([]github.Repository, error) {
	repos, err := s.githubClient.InstallationRepositories()
	if err != nil {
		return nil, fmt.Errorf("cannot get installation repositories: %v", err)
	}
	return repos, nil
}

func (s Server) pollRepos(ctx context.Context) {
	for {
		newRepos, err := s.refreshRepos()
		if err != nil {
			slog.Error("cannot refresh repositories", "error", err)
		}
		// enqueue repos not yet synced to their latest push, which also retries
		// previously failed syncs
		for _, repo := range newRepos {
			if s.syncState.upToDate(repo.Id, repo.PushedAt.Time) {
				continue
			}
			slog.Info("repo has new commits", "repo", repo.FullName, "pushed_at", repo.PushedAt)
			select {
			case s.updatedRepos <- &syncRequest{repo: &repo}:
			default:
				slog.Warn("sync queue full, dropping poll event", "repo", repo.FullName)
			}
		}

		select {
		case <-ctx.Done():
			slog.Info("stopping repo poller")
			return
		case <-time.After(10 * time.Minute):
		}
	}
}
