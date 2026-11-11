package main

import (
	"context"
	"os"
	"os/exec"
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

func TestEnsureSeedScopeAll(t *testing.T) {
	if _, err := exec.LookPath("rad"); err != nil {
		t.Skip("rad not in PATH")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	_ = os.Unsetenv("SSH_AUTH_SOCK")
	radCmd(t, home, "auth", "--alias", "test")
	rid := "rad:z2dqRKkK5yu89w3CMX2dVsYrRwvFk"
	radCmd(t, home, "seed", rid, "--scope", "followed", "--no-fetch")

	if err := ensureSeedScope(context.Background(), home, rid); err != nil {
		t.Fatal(err)
	}

	out := radCmd(t, home, "seed")
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, rid) {
			if !strings.Contains(line, " all ") {
				t.Fatalf("scope not updated: %s", line)
			}
			return
		}
	}
	t.Fatalf("rid missing from seed list:\n%s", out)
}
