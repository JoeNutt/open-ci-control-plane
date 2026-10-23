// SPDX-License-Identifier: GPL-3.0-or-later

package jsonl_test

import (
	"context"
	"path/filepath"
	"slices"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/history/jsonl"
)

func TestAnEscapeSurvivesARestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.jsonl")
	dispatched := dispatchedPlanOf(t)
	suite, err := domain.ParseCoverageSuite(ledgerAlwaysRun + "/other")
	if err != nil {
		t.Fatalf("parse suite: %v", err)
	}
	escape := domain.Escape{
		Mode: domain.ShadowedSelection, PullRequest: dispatched.Locator, ChangedFiles: dispatched.ChangedFiles,
		Test: domain.TestResult{Suite: suite, Name: "TestRefund", Outcome: domain.TestOutcomeFail, Rerun: domain.TestOutcomeFail},
	}

	if err := jsonl.NewEscapes(openedLog(t, path)).RecordEscape(context.Background(), escape); err != nil {
		t.Fatalf("record escape: %v", err)
	}

	escapes, err := jsonl.NewEscapes(openedLog(t, path)).RecordedEscapes()
	if err != nil {
		t.Fatalf("read escapes: %v", err)
	}
	if len(escapes) != 1 || escapes[0].Mode != escape.Mode || escapes[0].PullRequest != escape.PullRequest ||
		escapes[0].Test != escape.Test || !slices.Equal(escapes[0].ChangedFiles, escape.ChangedFiles) {
		t.Errorf("after a restart the escapes are %+v, want %+v", escapes, escape)
	}
}
