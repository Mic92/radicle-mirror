package main

import (
	"flag"
	"fmt"
	"strings"
	"time"
)

type Args struct {
	appId             int
	rsaKeyPath        string
	webhookSecretPath string

	radicleKey           string
	reposPath            string
	radHome              string
	githubEndpoint       string
	cloneHost            string
	addr                 string
	ridVarName           string
	mirroredForks        map[string]bool
	allowedOwners        map[string]bool
	delegates            []string
	radListen            []string
	radExternalAddresses []string
	workers              int
	syncTimeout          time.Duration
}

func splitList(s string) []string {
	out := []string{}
	for _, v := range strings.Split(s, ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func parseArgs() (*Args, error) {
	a := Args{}
	flag.IntVar(&a.appId, "gh-app-id", 0, "GitHub App ID")
	flag.StringVar(&a.rsaKeyPath, "gh-app-key-path", "", "Path to the GitHub App RSA private key file")
	flag.StringVar(&a.radicleKey, "radicle-key-path", "", "Path to the Radicle private key file")
	flag.StringVar(&a.reposPath, "repos-path", "./repos", "Path to the repositories directory")
	flag.StringVar(&a.radHome, "rad-home", "./radicle", "Path to the rad state")
	flag.StringVar(&a.githubEndpoint, "gh-endpoint", "https://api.github.com/", "GitHub endpoint to contact")
	flag.StringVar(&a.cloneHost, "gh-clone-host", "github.com", "Host that repositories may be cloned from over https")
	flag.StringVar(&a.ridVarName, "gh-rid-var-name", "RADICLE_RID", "Name of the environment variable to set with the repository name")

	var mirroredForks string
	flag.StringVar(&mirroredForks, "mirror-forks", "", "Comma-separated owner/repo names of forks to mirror; other forks are skipped")
	var delegates string
	flag.StringVar(&delegates, "delegate", "", "Comma-separated DIDs to add as delegates on mirrored repos")
	var allowedOwners string
	flag.StringVar(&allowedOwners, "allowed-owners", "", "Comma-separated GitHub users/orgs to mirror; empty allows all")
	var radListen, radExternal string
	flag.StringVar(&radListen, "rad-listen", "", "Comma-separated addresses the radicle node listens on for P2P connections")
	flag.StringVar(&radExternal, "rad-external-address", "", "Comma-separated external addresses the radicle node announces")
	flag.IntVar(&a.workers, "workers", 4, "Number of concurrent repository sync workers")
	flag.DurationVar(&a.syncTimeout, "sync-timeout", 30*time.Minute, "Timeout for a single repository sync")
	flag.StringVar(&a.addr, "addr", ":4128", "Port to listen on")
	flag.StringVar(&a.webhookSecretPath, "gh-webhook-secret-path", "", "Path to the webhook secret file")
	flag.Parse()
	a.mirroredForks = make(map[string]bool)
	for _, name := range strings.Split(mirroredForks, ",") {
		if name = strings.TrimSpace(name); name != "" {
			a.mirroredForks[name] = true
		}
	}
	a.delegates = splitList(delegates)
	a.allowedOwners = make(map[string]bool)
	for _, name := range splitList(allowedOwners) {
		a.allowedOwners[name] = true
	}
	a.radListen = splitList(radListen)
	a.radExternalAddresses = splitList(radExternal)
	if a.radicleKey == "" {
		return nil, fmt.Errorf("no --radicle-key-path set")
	}
	if a.appId == 0 {
		return nil, fmt.Errorf("no --gh-app-id set")
	}
	if a.rsaKeyPath == "" {
		return nil, fmt.Errorf("no --gh-app-key-path set")
	}
	if a.webhookSecretPath == "" {
		return nil, fmt.Errorf("no --gh-webhook-secret-path set")
	}
	if a.workers < 1 {
		return nil, fmt.Errorf("--workers must be at least 1")
	}

	return &a, nil
}
