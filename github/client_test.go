package github

import "testing"

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
			"/installation/repositories/?per_page=100&page=1",
			"https://api.github.com/installation/repositories/?per_page=100&page=1",
		},
	}
	for _, tt := range tests {
		if got := joinURL(tt.base, tt.path); got != tt.want {
			t.Errorf("joinURL(%q, %q) = %q, want %q", tt.base, tt.path, got, tt.want)
		}
	}
}
