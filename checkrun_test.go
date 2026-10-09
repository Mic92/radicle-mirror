package main

import (
	"errors"
	"strings"
	"testing"
)

// The summary must link to the radicle-mirror project so the status check
// advertises where it comes from.
func TestCheckRunSummaryLinksProject(t *testing.T) {
	success := buildCheckRun("https://example.com", "abc", "rad:z123", nil)
	if !strings.Contains(success.Output.Summary, projectURL) {
		t.Errorf("success summary lacks project link: %q", success.Output.Summary)
	}
	if success.Conclusion != "success" {
		t.Errorf("unexpected conclusion: %q", success.Conclusion)
	}

	failure := buildCheckRun("", "abc", "rad:z123", errors.New("boom"))
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

func TestCheckRunSummaryShowsRid(t *testing.T) {
	for _, err := range []error{nil, errors.New("boom")} {
		run := buildCheckRun("", "abc", "rad:z123", err)
		if !strings.Contains(run.Output.Summary, "rad:z123") {
			t.Errorf("summary lacks rid: %q", run.Output.Summary)
		}
	}
	run := buildCheckRun("", "abc", "", nil)
	if strings.Contains(run.Output.Summary, "RID") {
		t.Errorf("summary mentions RID without one: %q", run.Output.Summary)
	}
}

func TestCheckRunSummaryLinksExplorer(t *testing.T) {
	run := buildCheckRun("https://explorer.example/x", "abc", "rad:z123", nil)
	if !strings.Contains(run.Output.Summary, "(https://explorer.example/x)") {
		t.Errorf("summary lacks explorer link: %q", run.Output.Summary)
	}
	run = buildCheckRun("", "abc", "rad:z123", nil)
	if strings.Contains(run.Output.Summary, "explorer") {
		t.Errorf("summary has explorer link without URL: %q", run.Output.Summary)
	}
}
