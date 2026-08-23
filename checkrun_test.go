package main

import (
	"errors"
	"strings"
	"testing"
)

// The summary must link to the radicle-mirror project so the status check
// advertises where it comes from.
func TestCheckRunSummaryLinksProject(t *testing.T) {
	success := buildCheckRun("https://example.com", "abc", nil)
	if !strings.Contains(success.Output.Summary, projectURL) {
		t.Errorf("success summary lacks project link: %q", success.Output.Summary)
	}
	if success.Conclusion != "success" {
		t.Errorf("unexpected conclusion: %q", success.Conclusion)
	}

	failure := buildCheckRun("", "abc", errors.New("boom"))
	if !strings.Contains(failure.Output.Summary, projectURL) {
		t.Errorf("failure summary lacks project link: %q", failure.Output.Summary)
	}
	if !strings.Contains(failure.Output.Summary, "boom") {
		t.Errorf("failure summary lacks error: %q", failure.Output.Summary)
	}
	if failure.Conclusion != "failure" {
		t.Errorf("unexpected conclusion: %q", failure.Conclusion)
	}
}
