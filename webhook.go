package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/Mic92/radicle/github"
)

// only include the subset of data I need here

type PushEvent struct {
	Ref        string            `json:"ref"`
	After      string            `json:"after"`
	Repository github.Repository `json:"repository"`
}

func verifySignature(payloadBody []byte, signature string, secretToken string) bool {
	hmac := hmac.New(sha256.New, []byte(secretToken))
	hmac.Write(payloadBody)
	computedSignature := "sha256=" + hex.EncodeToString(hmac.Sum(nil))
	res := subtle.ConstantTimeCompare([]byte(computedSignature), []byte(signature))
	return res != 0
}

// maxWebhookBody caps the request body to bound memory use per delivery.
const maxWebhookBody = 25 << 20

func (s Server) githubHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxWebhookBody))
	if err != nil {
		slog.Warn("cannot read http body", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("failed to read http body"))
		return
	}

	if !verifySignature(body, r.Header.Get("X-Hub-Signature-256"), s.webhookSecret) {
		slog.Warn("invalid signature", "remote", r.RemoteAddr)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("invalid signature"))
		return
	}

	if event := r.Header.Get("X-GitHub-Event"); event != "push" {
		slog.Debug("ignoring non-push event", "event", event)
		w.WriteHeader(http.StatusOK)
		return
	}

	var pushEvent PushEvent
	err = json.Unmarshal(body, &pushEvent)
	if err != nil {
		slog.Warn("cannot decode http body", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("failed to decode http body as json"))
		return
	}

	select {
	case s.updatedRepos <- &syncRequest{repo: &pushEvent.Repository, headSha: pushEvent.After}:
	default:
		slog.Warn("sync queue full, dropping event", "repo", pushEvent.Repository.FullName)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("sync queue full"))
	}
}
