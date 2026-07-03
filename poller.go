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
	repoTimestamps := make(map[int]time.Time)
	for {
		newRepos, err := s.refreshRepos()
		if err != nil {
			slog.Error("cannot refresh repositories", "error", err)
		}
		// has updates
		for _, repo := range newRepos {
			if repo.PushedAt.After(repoTimestamps[repo.Id]) {
				slog.Info("repo has new commits", "repo", repo.FullName, "pushed_at", repo.PushedAt)
				repoTimestamps[repo.Id] = repo.PushedAt.Time
				s.updatedRepos <- &repo
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
