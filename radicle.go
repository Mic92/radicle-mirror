package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type RadNode struct {
	Command *exec.Cmd
	Home    string
}

// installKey places the provided OpenSSH ed25519 private key into the radicle
// key store and derives the matching public key, so the node runs under a
// stable, externally managed identity instead of a random generated one.
func installKey(home string, privateKeyPath string) error {
	keysPath := filepath.Join(home, "keys")
	if err := os.MkdirAll(keysPath, 0o700); err != nil {
		return fmt.Errorf("cannot create directory '%s': %w", keysPath, err)
	}
	privDst := filepath.Join(keysPath, "radicle")
	if _, err := os.Stat(privDst); err == nil {
		return nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("cannot stat '%s': %w", privDst, err)
	}
	key, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return fmt.Errorf("cannot read radicle key: %w", err)
	}
	if err := os.WriteFile(privDst, key, 0o600); err != nil {
		return fmt.Errorf("cannot write radicle key: %w", err)
	}
	pub, err := exec.Command("ssh-keygen", "-y", "-f", privDst).Output()
	if err != nil {
		return fmt.Errorf("cannot derive radicle public key: %w", err)
	}
	// radicle stores the public key with a "radicle" comment
	pubLine := strings.TrimRight(string(pub), "\n") + " radicle\n"
	if err := os.WriteFile(privDst+".pub", []byte(pubLine), 0o644); err != nil {
		return fmt.Errorf("cannot write radicle public key: %w", err)
	}
	return nil
}

func NewNode(home string, privateKeyPath string) (*RadNode, error) {
	env := os.Environ()
	env = append(env, fmt.Sprintf("RAD_HOME=%s", home))
	if err := installKey(home, privateKeyPath); err != nil {
		return nil, fmt.Errorf("cannot install radicle key: %w", err)
	}

	configPath := filepath.Join(home, "config.json")
	if _, err := os.Stat(configPath); errors.Is(err, fs.ErrNotExist) {
		init := exec.Command("rad", "config", "init", "--alias", "radicle-mirror")
		init.Env = env
		init.Stderr = os.Stderr
		if err := init.Run(); err != nil {
			return nil, fmt.Errorf("cannot initialize rad config: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("cannot stat '%s': %w", configPath, err)
	}

	cmd := exec.Command("rad", "node", "start", "--foreground")
	cmd.Env = env
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Dir = home

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start rad node: %w", err)
	}

	// wait for the control socket to appear so callers can safely use the node
	socket := filepath.Join(home, "node", "control.sock")
	for i := 0; i < 100; i++ {
		if _, err := os.Stat(socket); err == nil {
			return &RadNode{cmd, home}, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil, fmt.Errorf("rad node did not become ready within 10s")
}

func (n *RadNode) Stop() error {
	if n.Command == nil || n.Command.Process == nil {
		return nil
	}
	if err := n.Command.Process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("cannot signal rad node: %w", err)
	}
	// a clean shutdown via SIGTERM surfaces as a signal exit error
	var exitErr *exec.ExitError
	if err := n.Command.Wait(); err != nil && !errors.As(err, &exitErr) {
		return err
	}
	return nil
}

func radEnv(home string) []string {
	return append(os.Environ(), fmt.Sprintf("RAD_HOME=%s", home))
}

func getRadId(home string, repoPath string) (string, error) {
	cmd2 := exec.Command("rad", "inspect", "--rid")
	cmd2.Dir = repoPath
	cmd2.Env = radEnv(home)
	stderr := bytes.Buffer{}
	stdout := bytes.Buffer{}
	cmd2.Stderr = &stderr
	cmd2.Stdout = &stdout
	if err := cmd2.Run(); err != nil {
		return "", fmt.Errorf("rad inspect command failed: %w, output: %s", err, stderr.String())
	}
	rid := strings.TrimSpace(stdout.String())
	if rid == "" {
		return "", fmt.Errorf("rad inspect returned empty rid, stderr: %s", stderr.String())
	}
	return rid, nil
}

type RadMetadata struct {
	Home        string
	RepoPath    string
	Name        string
	Description string
	ExistingRad string
	Private     bool
}

func ensureRad(metadata RadMetadata) (string, error) {
	_ = exec.Command("git", "-C", metadata.RepoPath, "remote", "remove", "rad").Run()
	args := []string{
		"init",
		"--no-confirm",
		"--name", metadata.Name,
		"--description", metadata.Description,
	}
	if metadata.ExistingRad != "" {
		args = append(args, "--existing")
		args = append(args, metadata.ExistingRad)
	}
	if metadata.Private {
		args = append(args, "--private")
	} else {
		args = append(args, "--public")
	}
	cmd := exec.Command("rad", args...)
	cmd.Dir = metadata.RepoPath
	cmd.Env = radEnv(metadata.Home)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("rad init command failed: %w, stdout: %s, stderr: %s", err, stdout.String(), stderr.String())
	}
	return getRadId(metadata.Home, metadata.RepoPath)
}
