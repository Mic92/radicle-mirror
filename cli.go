package main

import (
	"flag"
	"fmt"
	"os"
)

type Args struct {
	appId             int
	rsaKeyPath        string
	webhookSecretPath string

	radicleKey     string
	reposPath      string
	radHome        string
	githubEndpoint string
	addr           string
	ridVarName     string
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Options:\n")
	fmt.Fprintf(os.Stderr, "  --addr string\n")
	fmt.Fprintf(os.Stderr, "        Port to listen on (default \":4128\")\n")
	fmt.Fprintf(os.Stderr, "  --radicle-key-path string\n")
	fmt.Fprintf(os.Stderr, "        Path to the Radicle private key file\n")
	fmt.Fprintf(os.Stderr, "  --repos-path string\n")
	fmt.Fprintf(os.Stderr, "        Path to the repositories directory\n")
	fmt.Fprintf(os.Stderr, "  --rad-home string\n")
	fmt.Fprintf(os.Stderr, "        Path to the rad state\n")

	fmt.Fprintf(os.Stderr, "  --gh-app-id int\n")
	fmt.Fprintf(os.Stderr, "        GitHub App ID\n")
	fmt.Fprintf(os.Stderr, "  --gh-app-key-path string\n")
	fmt.Fprintf(os.Stderr, "        Path to the GitHub App RSA private key file\n")
	fmt.Fprintf(os.Stderr, "  --gh-endpoint string\n")
	fmt.Fprintf(os.Stderr, "        GitHub endpoint to contact")
	fmt.Fprintf(os.Stderr, "  --gh-rid-var-name string\n")
	fmt.Fprintf(os.Stderr, "        Name of the environment variable to set with the repository name (default \"RADICLE_RID\")\n")
	fmt.Fprintf(os.Stderr, "  --gh-webhook-secret-path string\n")
	fmt.Fprintf(os.Stderr, "        Path to the webhook secret file\n")

	fmt.Fprintf(os.Stderr, "  --help\n")
	fmt.Fprintf(os.Stderr, "        Show this help message\n")
}

func parseArgs() (*Args, error) {
	a := Args{}
	flag.Usage = printUsage
	flag.IntVar(&a.appId, "gh-app-id", 0, "GitHub App ID")
	flag.StringVar(&a.rsaKeyPath, "gh-app-key-path", "", "Path to the GitHub App RSA private key file")
	flag.StringVar(&a.radicleKey, "radicle-key-path", "", "Path to the Radicle private key file")
	flag.StringVar(&a.reposPath, "repos-path", "./repos", "Path to the repositories directory")
	flag.StringVar(&a.radHome, "rad-home", "./radicle", "Path to the rad state")
	flag.StringVar(&a.githubEndpoint, "gh-endpoint", "https://api.github.com/", "GitHub endpoint to contact")
	flag.StringVar(&a.ridVarName, "gh-rid-var-name", "RADICLE_RID", "Name of the environment variable to set with the repository name")

	flag.StringVar(&a.addr, "addr", ":4128", "Port to listen on")
	flag.StringVar(&a.webhookSecretPath, "gh-webhook-secret-path", "", "Path to the webhook secret file")
	flag.Parse()
	if a.radicleKey == "" {
		return nil, fmt.Errorf("No --radicle-key-path set")
	}

	if a.appId == 0 {
		return nil, fmt.Errorf("No --gh-app-id set")
	}
	if a.rsaKeyPath == "" {
		return nil, fmt.Errorf("No --gh-app-key-path set")
	}
	if a.webhookSecretPath == "" {
		return nil, fmt.Errorf("No --gh-webhook-secret-path set")
	}

	return &a, nil
}
