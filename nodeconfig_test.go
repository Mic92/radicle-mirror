package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateNodeConfig(t *testing.T) {
	home := t.TempDir()
	cfg := map[string]any{
		"publicExplorer": "https://app.radicle.xyz",
		"node": map[string]any{
			"alias":             "radicle-mirror",
			"listen":            []any{},
			"externalAddresses": []any{},
		},
	}
	b, _ := json.Marshal(cfg)
	path := filepath.Join(home, "config.json")
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}

	listen := []string{"0.0.0.0:8776"}
	external := []string{"radicle.example.com:8776"}
	if err := updateNodeConfig(home, listen, external); err != nil {
		t.Fatal(err)
	}

	var got map[string]any
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	node := got["node"].(map[string]any)
	if node["listen"].([]any)[0] != "0.0.0.0:8776" {
		t.Errorf("listen not updated: %v", node["listen"])
	}
	if node["externalAddresses"].([]any)[0] != "radicle.example.com:8776" {
		t.Errorf("externalAddresses not updated: %v", node["externalAddresses"])
	}
	if node["alias"] != "radicle-mirror" {
		t.Errorf("unrelated key clobbered: %v", node["alias"])
	}
	if got["publicExplorer"] != "https://app.radicle.xyz" {
		t.Errorf("unrelated top-level key clobbered: %v", got["publicExplorer"])
	}
}
