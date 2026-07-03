package main

// Syncs external git repos <-> Radicle

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/Mic92/radicle/github"
)

type SyncState map[int]time.Time

func (s *Server) runGitCommand(args ...string) error {
	credsHelper := "credential.helper=!f() { echo \"username=token\"; echo \"password=$GITHUB_TOKEN\"; }; f"
	args = append([]string{"-c", credsHelper}, args...)
	env := os.Environ()
	env = append(env, "GIT_TERMINAL_PROMPT=0")
	// GITHUB_TOKEN
	token, err := s.githubClient.Token()
	if err != nil {
		return fmt.Errorf("cannot get github token: %w", err)
	}
	env = append(env, "GITHUB_TOKEN="+token)
	cmd := exec.Command("git", args...)
	cmd.Env = env
	// capture stderr
	var buf bytes.Buffer
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cannot start git command: %w, output: %s", err, buf.String())
	}
	return nil
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func pushRadRepo(home string, repoPath string) error {
	cmd := exec.Command("git", "push", "--mirror", "rad")
	cmd.Dir = repoPath
	cmd.Env = radEnv(home)
	buf := bytes.Buffer{}
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rad push command failed: %w, output: %s", err, buf.String())
	}
	return nil
}

func (s *Server) syncRepo(repo *github.Repository, syncState SyncState) error {
	pushedAt, ok := syncState[repo.Id]
	if ok {
		if pushedAt.After(repo.PushedAt.Time) {
			slog.Debug("repo is up to date, skipping", "repo", repo)
			return nil
		}
	} else {
		slog.Info("syncing new repo", "repo", repo)
	}
	radId, err := s.githubClient.GetRepoVar(repo.Owner.Login, repo.Name, s.repoVarName, "")
	if err != nil {
		return fmt.Errorf("cannot get repo var: %w", err)
	}
	// run git to fetch the latest changes and update the radicle repo to s.reposPath
	repoPath := filepath.Join(s.reposPath, strconv.Itoa(repo.Owner.Id), strconv.Itoa(repo.Id))
	exists, err := pathExists(repoPath)
	if err != nil {
		return fmt.Errorf("cannot check if repo path exists: %w", err)
	} else if !exists {
		if err := os.MkdirAll(repoPath, 0o755); err != nil {
			return fmt.Errorf("cannot create repo path: %w", err)
		}
		// use https token to clone the repo
		err := s.runGitCommand("clone", "--mirror", repo.CloneUrl, repoPath)
		if err != nil {
			return fmt.Errorf("cannot clone git command: %w", err)
		}
		slog.Info("cloning repo", "repo", repo)
	} else {
		err := s.runGitCommand("-C", repoPath, "remote", "set-url", "origin", repo.CloneUrl)
		if err != nil {
			return fmt.Errorf("cannot set git origin: %w", err)
		}
		err = s.runGitCommand("-C", repoPath, "remote", "update", "--prune")
		if err != nil {
			return fmt.Errorf("cannot pull repository %s: %w", repo.CloneUrl, err)
		}
	}
	metadata := RadMetadata{
		Home:        s.radHome,
		RepoPath:    repoPath,
		Name:        repo.Name,
		Description: repo.Description,
		ExistingRad: radId,
		Private:     repo.Private,
	}
	newRadId, err := ensureRad(metadata)
	if err != nil {
		return fmt.Errorf("cannot initialize rad remote: %w", err)
	}
	if newRadId != radId {
		err = s.githubClient.SetRepoVar(repo.Owner.Login, repo.Name, s.repoVarName, newRadId)
		if err != nil {
			return fmt.Errorf("cannot set repo var: %w", err)
		}
	}
	// push the changes to radicle
	if err := pushRadRepo(s.radHome, repoPath); err != nil {
		return fmt.Errorf("cannot push radicle repo: %w", err)
	}
	syncState[repo.Id] = repo.PushedAt.Time
	return nil
}

func (s *Server) syncer(ctx context.Context) {
	syncState := make(SyncState)
	for {
		select {
		case <-ctx.Done():
			slog.Info("stopping syncer")
			return
		case repo := <-s.updatedRepos:
			slog.Info("syncing repo", "repo", repo)
			if err := s.syncRepo(repo, syncState); err != nil {
				slog.Error("cannot sync repo", "repo", repo, "error", err)
			}
		}
	}
}
