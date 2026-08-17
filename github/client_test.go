package github

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestJoinURL(t *testing.T) {
	tests := []struct {
		base string
		path string
		want string
	}{
		{"https://api.github.com/", "/app/installations", "https://api.github.com/app/installations"},
		{"https://api.github.com", "app/installations", "https://api.github.com/app/installations"},
		{
			"https://api.github.com/",
			"/installation/repositories?per_page=100&page=1",
			"https://api.github.com/installation/repositories?per_page=100&page=1",
		},
	}
	for _, tt := range tests {
		if got := joinURL(tt.base, tt.path); got != tt.want {
			t.Errorf("joinURL(%q, %q) = %q, want %q", tt.base, tt.path, got, tt.want)
		}
	}
}

func TestPerInstallationTokens(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(t.TempDir(), "key.pem")
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	if err := os.WriteFile(keyPath, pemBytes, 0o600); err != nil {
		t.Fatal(err)
	}

	var checkRunAuth string
	mux := http.NewServeMux()
	mux.HandleFunc("/app/installations", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[
			{"id": 1, "app_id": 42, "account": {"login": "numtide"}},
			{"id": 2, "app_id": 42, "account": {"login": "Mic92"}}
		]`)
	})
	mux.HandleFunc("/app/installations/1/access_tokens", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"token": "token-numtide"}`)
	})
	mux.HandleFunc("/app/installations/2/access_tokens", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"token": "token-mic92"}`)
	})
	mux.HandleFunc("/repos/Mic92/dotfiles/check-runs", func(w http.ResponseWriter, r *http.Request) {
		checkRunAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusCreated)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c, err := NewClient(srv.URL, 42, keyPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.CreateCheckRun("Mic92", "dotfiles", CheckRun{Name: "x", HeadSha: "abc"}); err != nil {
		t.Fatal(err)
	}
	if checkRunAuth != "Bearer token-mic92" {
		t.Errorf("check run used wrong token: %q", checkRunAuth)
	}
}
