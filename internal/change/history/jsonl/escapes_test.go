// SPDX-License-Identifier: GPL-3.0-or-later

package jsonl_test

import (
	"context"
	"encoding/json"
	"errors"
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

	escapes, err := jsonl.NewEscapes(openedLog(t, path)).RecordedEscapes(context.Background())
	if err != nil {
		t.Fatalf("read escapes: %v", err)
	}
	if len(escapes) != 1 || escapes[0].Mode != escape.Mode || escapes[0].PullRequest != escape.PullRequest ||
		escapes[0].Test != escape.Test || !slices.Equal(escapes[0].ChangedFiles, escape.ChangedFiles) {
		t.Errorf("after a restart the escapes are %+v, want %+v", escapes, escape)
	}
}

func TestAnEscapeLineNamingAModeThatIsNoModeIsRefused(t *testing.T) {
	history := openedLog(t, filepath.Join(t.TempDir(), "history.jsonl"))
	hostile := `{"kind":"escape","mode":"everything","repository":"` + ledgerRepository + `","pullRequest":7,` +
		`"suite":"` + ledgerSuite + `","test":"TestRefund","outcome":"fail","rerun":"fail","changedFiles":[]}`
	if err := history.Append(json.RawMessage(hostile)); err != nil {
		t.Fatalf("append: %v", err)
	}

	_, err := jsonl.NewEscapes(history).RecordedEscapes(context.Background())

	if !errors.Is(err, domain.ErrUnknownShadowedMode) {
		t.Errorf("an escape line with a mode that is no mode gave %v, want %v", err, domain.ErrUnknownShadowedMode)
	}
}

func TestARecordedEscapeCarriesTheTimeItWasRecorded(t *testing.T) {
	escapes := jsonl.NewEscapes(openedLog(t, filepath.Join(t.TempDir(), "history.jsonl")))
	suite, err := domain.ParseCoverageSuite(ledgerSuite)
	if err != nil {
		t.Fatalf("parse suite: %v", err)
	}
	escape := domain.Escape{
		Mode: domain.ShadowedSelection, PullRequest: dispatchedPlanOf(t).Locator,
		Test: domain.TestResult{Suite: suite, Name: "TestRefund", Outcome: domain.TestOutcomeFail, Rerun: domain.TestOutcomeFail},
	}
	if err := escapes.RecordEscape(context.Background(), escape); err != nil {
		t.Fatalf("record escape: %v", err)
	}

	recorded, err := escapes.RecordedEscapes(context.Background())

	if err != nil || len(recorded) != 1 || !recorded[0].At.Equal(recordedAt) {
		t.Errorf("the escape carries %+v (%v), want the time it was recorded, %v", recorded, err, recordedAt)
	}
}
