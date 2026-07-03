package main

// Syncs external git repos <-> Radicle

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Mic92/radicle/github"
)

const (
	maxSyncRetries = 5
	syncRetryBase  = 30 * time.Second
)

// syncState records the last successfully synced pushedAt per repo. Failures
// leave the entry untouched, so the poller re-enqueues them.
type syncState struct {
	mu         sync.Mutex
	lastSynced map[int]time.Time
}

func newSyncState() *syncState {
	return &syncState{lastSynced: make(map[int]time.Time)}
}

func (s *syncState) upToDate(id int, pushedAt time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	last, ok := s.lastSynced[id]
	return ok && !pushedAt.After(last)
}

func (s *syncState) markSynced(id int, pushedAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastSynced[id] = pushedAt
}

// validateCloneURL blocks non-https transports (ext::, file::) that git would
// execute and hosts the credential helper must not leak the token to.
func (s *Server) validateCloneURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("cannot parse clone url %q: %w", rawURL, err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("clone url %q does not use https", rawURL)
	}
	if u.Hostname() != s.cloneHost {
		return fmt.Errorf("clone url host %q is not the allowed host %q", u.Hostname(), s.cloneHost)
	}
	return nil
}

func (s *Server) runGitCommand(ctx context.Context, args ...string) error {
	// scope the helper to the clone host so the token never leaks to another host
	credsHelper := fmt.Sprintf("credential.https://%s.helper=!f() { echo \"username=token\"; echo \"password=$GITHUB_TOKEN\"; }; f", s.cloneHost)
	args = append([]string{"-c", credsHelper}, args...)
	env := os.Environ()
	env = append(env, "GIT_TERMINAL_PROMPT=0")
	// GITHUB_TOKEN
	token, err := s.githubClient.Token()
	if err != nil {
		return fmt.Errorf("cannot get github token: %w", err)
	}
	env = append(env, "GITHUB_TOKEN="+token)
	cmd := exec.CommandContext(ctx, "git", args...)
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

func pushRadRepo(ctx context.Context, home string, repoPath string) error {
	cmd := exec.CommandContext(ctx, "git", "push", "--mirror", "rad")
	cmd.Dir = repoPath
	cmd.Env = radEnv(home)
	buf := bytes.Buffer{}
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rad push command failed: %w, output: %s", err, buf.String())
	}
	return nil
}

func (s *Server) syncRepo(ctx context.Context, repo *github.Repository) error {
	if s.syncState.upToDate(repo.Id, repo.PushedAt.Time) {
		slog.Debug("repo is up to date, skipping", "repo", repo)
		return nil
	}
	if err := s.validateCloneURL(repo.CloneUrl); err != nil {
		return err
	}
	// run git to fetch the latest changes and update the radicle repo to s.reposPath
	repoPath := filepath.Join(s.reposPath, strconv.Itoa(repo.Owner.Id), strconv.Itoa(repo.Id))
	// local .rid file is authoritative; GitHub variable is only a recovery fallback
	ridPath := repoPath + ".rid"
	radId := readRid(ridPath)
	if radId == "" {
		radId, _ = s.githubClient.GetRepoVar(repo.Owner.Login, repo.Name, s.repoVarName, "")
	}
	exists, err := pathExists(repoPath)
	if err != nil {
		return fmt.Errorf("cannot check if repo path exists: %w", err)
	} else if !exists {
		if err := os.MkdirAll(repoPath, 0o755); err != nil {
			return fmt.Errorf("cannot create repo path: %w", err)
		}
		// use https token to clone the repo
		err := s.runGitCommand(ctx, "clone", "--mirror", repo.CloneUrl, repoPath)
		if err != nil {
			return fmt.Errorf("cannot clone git command: %w", err)
		}
		slog.Info("cloning repo", "repo", repo)
	} else {
		err := s.runGitCommand(ctx, "-C", repoPath, "remote", "set-url", "origin", repo.CloneUrl)
		if err != nil {
			return fmt.Errorf("cannot set git origin: %w", err)
		}
		// fetch only origin: "remote update" would also fetch the rad remote,
		// invoking git-remote-rad without RAD_HOME set on this command
		err = s.runGitCommand(ctx, "-C", repoPath, "fetch", "--prune", "origin")
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
	newRadId, err := ensureRad(ctx, metadata)
	if err != nil {
		return fmt.Errorf("cannot initialize rad remote: %w", err)
	}
	if newRadId != radId {
		if err := os.WriteFile(ridPath, []byte(newRadId+"\n"), 0o644); err != nil {
			return fmt.Errorf("cannot persist rad id: %w", err)
		}
		// best-effort publish for visibility; not required for mirroring
		if err := s.githubClient.SetRepoVar(repo.Owner.Login, repo.Name, s.repoVarName, newRadId); err != nil {
			slog.Warn("cannot publish rad id to github", "repo", repo.FullName, "error", err)
		}
	}
	// push the changes to radicle
	if err := pushRadRepo(ctx, s.radHome, repoPath); err != nil {
		return fmt.Errorf("cannot push radicle repo: %w", err)
	}
	s.syncState.markSynced(repo.Id, repo.PushedAt.Time)
	return nil
}

func readRid(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func (s *Server) reportCheckRun(repo *github.Repository, headSha string, syncErr error) error {
	run := github.CheckRun{
		Name:       "radicle-mirror",
		HeadSha:    headSha,
		Status:     "completed",
		Conclusion: "success",
		Output: github.CheckRunOutput{
			Title:   "Radicle mirror",
			Summary: "Repository mirrored to Radicle.",
		},
	}
	if syncErr != nil {
		run.Conclusion = "failure"
		run.Output.Summary = fmt.Sprintf("Mirror to Radicle failed: %s", syncErr)
	}
	return s.githubClient.CreateCheckRun(repo.Owner.Login, repo.Name, run)
}

// syncer shards requests to a worker pool by repo id: serial per repo, concurrent
// across repos, so a slow repo cannot block the others.
func (s *Server) syncer(ctx context.Context) {
	var wg sync.WaitGroup
	shards := make([]chan *syncRequest, s.workers)
	for i := range shards {
		shards[i] = make(chan *syncRequest, 1024)
		wg.Add(1)
		go func(ch chan *syncRequest) {
			defer wg.Done()
			for req := range ch {
				s.handleSync(ctx, req)
			}
		}(shards[i])
	}

	for {
		select {
		case <-ctx.Done():
			slog.Info("stopping syncer")
			for _, ch := range shards {
				close(ch)
			}
			wg.Wait()
			return
		case req := <-s.updatedRepos:
			shard := shards[uint(req.repo.Id)%uint(len(shards))]
			select {
			case shard <- req:
			default:
				slog.Warn("sync shard full, dropping event", "repo", req.repo.FullName)
			}
		}
	}
}

func (s *Server) handleSync(ctx context.Context, req *syncRequest) {
	slog.Info("syncing repo", "repo", req.repo, "attempt", req.attempts+1)
	syncCtx, cancel := context.WithTimeout(ctx, s.syncTimeout)
	err := s.syncRepo(syncCtx, req.repo)
	cancel()

	if err != nil {
		slog.Error("cannot sync repo", "repo", req.repo, "error", err)
		// retry transient failures with exponential backoff unless shutting down
		if ctx.Err() == nil && req.attempts+1 < maxSyncRetries {
			retry := *req
			retry.attempts++
			delay := syncRetryBase << req.attempts
			time.AfterFunc(delay, func() {
				select {
				case s.updatedRepos <- &retry:
				default:
					slog.Warn("sync queue full, dropping retry", "repo", retry.repo.FullName)
				}
			})
			return
		}
	}

	// report the final outcome (success, or the last failed attempt)
	if req.headSha != "" {
		if reportErr := s.reportCheckRun(req.repo, req.headSha, err); reportErr != nil {
			slog.Error("cannot report check run", "repo", req.repo, "error", reportErr)
		}
	}
}
