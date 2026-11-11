package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func radCmd(t *testing.T, home string, args ...string) string {
	t.Helper()
	cmd := exec.Command("rad", args...)
	cmd.Env = append(radEnv(home), "RAD_PASSPHRASE=")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("rad %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func seedScopes(t *testing.T, home string) map[string]string {
	t.Helper()
	scopes := map[string]string{}
	for _, line := range strings.Split(radCmd(t, home, "seed"), "\n") {
		fields := strings.Fields(strings.Trim(line, "│ "))
		if len(fields) >= 2 && strings.HasPrefix(fields[0], "rad:") {
			scopes[fields[0]] = fields[len(fields)-1]
		}
	}
	return scopes
}

// Mirrors that are already up to date never go through syncRepo, so a stale
// 'followed' scope must be fixed on startup.
func TestReconcileSeedScopes(t *testing.T) {
	if _, err := exec.LookPath("rad"); err != nil {
		t.Skip("rad not in PATH")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	_ = os.Unsetenv("SSH_AUTH_SOCK")
	radCmd(t, home, "auth", "--alias", "test")

	reposPath := t.TempDir()
	rids := map[string]string{
		"Mic92/1": "rad:z2dqRKkK5yu89w3CMX2dVsYrRwvFk",
		"other/2": "rad:zaCSBVa8UbKNEWBcmRTW1m9fZXhu",
	}
	for path, rid := range rids {
		p := filepath.Join(reposPath, path)
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p+".rid", []byte(rid+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		radCmd(t, home, "seed", rid, "--scope", "followed", "--no-fetch")
	}

	s := &Server{radHome: home, reposPath: reposPath}
	if err := s.reconcileSeedScopes(context.Background()); err != nil {
		t.Fatal(err)
	}

	scopes := seedScopes(t, home)
	for _, rid := range rids {
		if scopes[rid] != "all" {
			t.Errorf("%s: scope %q, want all", rid, scopes[rid])
		}
	}
}
