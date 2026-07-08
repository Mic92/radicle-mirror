package main

import (
	"context"
	"log/slog"
	"time"
)

// pollRepos polls github repositories for new commits (in case a webhook was missed).
func (s Server) pollRepos(ctx context.Context) {
	for {
		newRepos, err := s.githubClient.InstallationRepositories()
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
