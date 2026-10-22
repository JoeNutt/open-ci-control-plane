// SPDX-License-Identifier: GPL-3.0-or-later

package jsonl_test

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/history/jsonl"
)

var recordedAt = time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)

func TestAnEntrySurvivesARestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.jsonl")
	written := json.RawMessage(`{"pullRequest":7,"escapes":0}`)

	before, err := jsonl.OpenLog(path, domain.DeclaredHistoryLimits(), func() time.Time { return recordedAt })
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := before.Append(written); err != nil {
		t.Fatalf("append: %v", err)
	}

	after, err := jsonl.OpenLog(path, domain.DeclaredHistoryLimits(), func() time.Time { return recordedAt })
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	entries, err := after.Entries()
	if err != nil {
		t.Fatalf("entries: %v", err)
	}
	if len(entries) != 1 || string(entries[0].Body) != string(written) || !entries[0].At.Equal(recordedAt) {
		t.Errorf("after a restart the log holds %+v, want one entry %s at %s", entries, written, recordedAt)
	}
}

func TestAnAppendThatWouldExceedTheBoundIsRefusedAndWritesNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.jsonl")
	body := json.RawMessage(`{"pullRequest":7}`)
	lineBytes := int64(len(`{"at":"2026-10-22T12:00:00Z","body":{"pullRequest":7}}`) + 1)
	history, err := jsonl.OpenLog(path, domain.HistoryLimits{MaxBytes: lineBytes}, func() time.Time { return recordedAt })
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	if err := history.Append(body); err != nil {
		t.Fatalf("an append at the limit was refused: %v", err)
	}
	if err := history.Append(body); !errors.Is(err, domain.ErrHistoryFull) {
		t.Errorf("an append one line over the limit gave %v, want %v", err, domain.ErrHistoryFull)
	}
	if entries, _ := history.Entries(); len(entries) != 1 {
		t.Errorf("the log holds %d entries after a refused append, want 1", len(entries))
	}
}

func TestAnEntryLargerThanTheScannersDefaultLineReadsBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.jsonl")
	large := json.RawMessage(`"` + strings.Repeat("x", 100<<10) + `"`)
	history, err := jsonl.OpenLog(path, domain.DeclaredHistoryLimits(), func() time.Time { return recordedAt })
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := history.Append(large); err != nil {
		t.Fatalf("append: %v", err)
	}

	entries, err := history.Entries()

	if err != nil || len(entries) != 1 || len(entries[0].Body) != len(large) {
		t.Errorf("reading a %d-byte entry back gave %d entries and %v", len(large), len(entries), err)
	}
}
