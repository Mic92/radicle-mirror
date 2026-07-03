package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestVerifySignature(t *testing.T) {
	body := []byte("hello world")
	secret := "topsecret"
	sig := sign(body, secret)
	if !verifySignature(body, sig, secret) {
		t.Error("valid signature rejected")
	}
	if verifySignature(body, sig, "wrong") {
		t.Error("signature accepted with wrong secret")
	}
	if verifySignature(body, "sha256=deadbeef", secret) {
		t.Error("forged signature accepted")
	}
}

func TestGithubHandler(t *testing.T) {
	body, err := os.ReadFile("gh_pr_event.json")
	if err != nil {
		t.Fatalf("cannot read fixture: %v", err)
	}
	secret := "topsecret"
	s := Server{
		webhookSecret: secret,
		updatedRepos:  make(chan *syncRequest, 1),
	}

	req := httptest.NewRequest("POST", "/github", strings.NewReader(string(body)))
	req.Header.Set("X-Hub-Signature-256", sign(body, secret))
	req.Header.Set("X-GitHub-Event", "push")
	rec := httptest.NewRecorder()
	s.githubHandler(rec, req)

	if rec.Code != 200 {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	select {
	case req := <-s.updatedRepos:
		if req.repo.FullName != "numtide/llm-agents.nix" {
			t.Errorf("unexpected repo: %q", req.repo.FullName)
		}
		if req.headSha != "0000000000000000000000000000000000000000" {
			t.Errorf("unexpected head sha: %q", req.headSha)
		}
	default:
		t.Fatal("no repo queued")
	}
}

func TestGithubHandlerInvalidSignature(t *testing.T) {
	body, err := os.ReadFile("gh_pr_event.json")
	if err != nil {
		t.Fatalf("cannot read fixture: %v", err)
	}
	s := Server{
		webhookSecret: "topsecret",
		updatedRepos:  make(chan *syncRequest, 1),
	}
	req := httptest.NewRequest("POST", "/github", strings.NewReader(string(body)))
	req.Header.Set("X-Hub-Signature-256", "sha256=bad")
	rec := httptest.NewRecorder()
	s.githubHandler(rec, req)

	if rec.Code != 401 {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if len(s.updatedRepos) != 0 {
		t.Error("repo queued despite invalid signature")
	}
}
