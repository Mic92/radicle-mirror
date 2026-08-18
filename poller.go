package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/Mic92/radicle/github"
)

func (s Server) warnUntracked(repos []github.Repository, reported map[string]bool) {
	known := make(map[int]bool, len(repos))
	for _, repo := range repos {
		known[repo.Id] = true
	}
	untracked, err := untrackedRepos(s.reposPath, known)
	if err != nil {
		slog.Error("cannot scan for untracked repos", "error", err)
		return
	}
	for _, u := range untracked {
		if !reported[u.Path] {
			reported[u.Path] = true
			slog.Warn("local mirror no longer exists on github", "path", u.Path, "rid", u.Rid)
		}
	}
}

// pollRepos polls github repositories for new commits (in case a webhook was missed).
func (s Server) pollRepos(ctx context.Context) {
	reported := make(map[string]bool)
	for {
		newRepos, err := s.githubClient.InstallationRepositories()
		if err != nil {
			slog.Error("cannot refresh repositories", "error", err)
		}
		s.warnUntracked(newRepos, reported)
		// enqueue repos not yet synced to their latest push, which also retries
		// previously failed syncs
		for _, repo := range newRepos {
			// filtered here as well to avoid re-enqueueing skipped forks on
			// every poll cycle
			if s.skipRepo(&repo) || s.syncState.upToDate(repo.Id, repo.PushedAt.Time) {
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
