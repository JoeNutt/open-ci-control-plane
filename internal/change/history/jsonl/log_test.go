// SPDX-License-Identifier: GPL-3.0-or-later

package jsonl_test

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/history/jsonl"
)

var recordedAt = time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)

func TestAnEntrySurvivesARestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.jsonl")
	written := json.RawMessage(`{"pullRequest":7,"escapes":0}`)

	before, err := jsonl.OpenLog(path, func() time.Time { return recordedAt })
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := before.Append(written); err != nil {
		t.Fatalf("append: %v", err)
	}

	after, err := jsonl.OpenLog(path, func() time.Time { return recordedAt })
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
