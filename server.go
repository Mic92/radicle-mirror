package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Mic92/radicle/github"
)

// headSha is set only for webhook pushes, enabling a check run for the commit.
type syncRequest struct {
	repo     *github.Repository
	headSha  string
	attempts int
}

type Server struct {
	webhookSecret string
	reposPath     string
	radHome       string
	cloneHost     string
	githubClient  *github.Client
	repoVarName   string
	updatedRepos  chan *syncRequest
	syncState     *syncState
	workers       int
	syncTimeout   time.Duration
}

func (s Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("ok"))
	if err != nil {
		slog.Warn("cannot write http response", "error", err)
	}
}

func runServer(args *Args) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	webhookSecret, err := os.ReadFile(args.webhookSecretPath)
	if err != nil {
		return fmt.Errorf("cannot read webhook secret: %v", err)
	}
	if len(bytes.TrimSpace(webhookSecret)) == 0 {
		return fmt.Errorf("webhook secret is empty")
	}

	s := Server{
		webhookSecret: string(webhookSecret),
		reposPath:     args.reposPath,
		radHome:       args.radHome,
		cloneHost:     args.cloneHost,
		repoVarName:   args.ridVarName,
		updatedRepos:  make(chan *syncRequest, 10000),
		syncState:     newSyncState(),
		workers:       args.workers,
		syncTimeout:   args.syncTimeout,
	}

	node, err := NewNode(args.radHome, args.radicleKey)
	if err != nil {
		return fmt.Errorf("cannot start rad node: %v", err)
	}
	defer func() {
		if err := node.Stop(); err != nil {
			slog.Error("cannot stop rad node", "error", err)
		}
	}()

	githubClient, err := github.NewClient(args.githubEndpoint, args.appId, args.rsaKeyPath)
	if err != nil {
		return fmt.Errorf("cannot create github client: %v", err)
	}
	s.githubClient = githubClient

	go s.pollRepos(ctx)
	go s.syncer(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/github", s.githubHandler)
	mux.HandleFunc("/health", s.healthHandler)

	srv := &http.Server{
		Addr:              args.addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, c := context.WithTimeout(context.Background(), 10*time.Second)
		defer c()
		err := srv.Shutdown(shutdownCtx)
		if err != nil {
			slog.Error("cannot shutdown http server", "error", err)
		}
	}()

	return srv.ListenAndServe()
}

func main() {
	args, err := parseArgs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	err = runServer(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
