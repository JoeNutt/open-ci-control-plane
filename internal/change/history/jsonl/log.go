// SPDX-License-Identifier: GPL-3.0-or-later

package jsonl

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type Entry struct {
	At   time.Time       `json:"at"`
	Body json.RawMessage `json:"body"`
}

type Log struct {
	guard  sync.Mutex
	path   string
	limits domain.HistoryLimits
	now    func() time.Time
}

func OpenLog(path string, limits domain.HistoryLimits, now func() time.Time) (*Log, error) {
	history := &Log{path: path, limits: limits, now: now}
	if err := history.pruneExpired(); err != nil {
		return nil, err
	}
	return history, nil
}

func (history *Log) pruneExpired() error {
	entries, isTorn, err := history.readEntries()
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	cutoff := history.now().Add(-history.limits.Retention)
	held := len(entries)
	entries = slices.DeleteFunc(entries, func(entry Entry) bool { return entry.At.Before(cutoff) })
	if len(entries) == held && !isTorn {
		return nil
	}
	return history.rewrite(entries)
}

func (history *Log) rewrite(entries []Entry) error {
	staged, err := os.CreateTemp(filepath.Dir(history.path), ".history-*")
	if err != nil {
		return fmt.Errorf("stage the pruned history: %w", err)
	}
	if err := writeAll(staged, entries); err != nil {
		return errors.Join(err, staged.Close(), os.Remove(staged.Name()))
	}
	return os.Rename(staged.Name(), history.path)
}

func writeAll(staged *os.File, entries []Entry) error {
	for _, entry := range entries {
		record, err := recordOf(entry)
		if err != nil {
			return err
		}
		if _, err := staged.Write(record); err != nil {
			return fmt.Errorf("write the pruned history: %w", err)
		}
	}
	return errors.Join(staged.Sync(), staged.Close())
}

func recordOf(entry Entry) ([]byte, error) {
	line, err := json.Marshal(entry)
	if err != nil {
		return nil, fmt.Errorf("encode a history entry: %w", err)
	}
	return append(line, '\n'), nil
}

func (history *Log) Append(body json.RawMessage) error {
	history.guard.Lock()
	defer history.guard.Unlock()

	record, err := recordOf(Entry{At: history.now(), Body: body})
	if err != nil {
		return err
	}
	file, err := os.OpenFile(history.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open the history: %w", err)
	}
	return errors.Join(history.writeWithinBound(file, record), file.Close())
}

func (history *Log) writeWithinBound(file *os.File, record []byte) error {
	held, err := file.Stat()
	if err != nil {
		return fmt.Errorf("measure the history: %w", err)
	}
	if held.Size()+int64(len(record)) > history.limits.MaxBytes {
		return domain.ErrHistoryFull
	}
	if _, err := file.Write(record); err != nil {
		return fmt.Errorf("write the history: %w", err)
	}
	return file.Sync()
}

func (history *Log) Entries() ([]Entry, error) {
	history.guard.Lock()
	defer history.guard.Unlock()

	entries, _, err := history.readEntries()
	return entries, err
}

func (history *Log) readEntries() (entries []Entry, isTorn bool, err error) {
	held, err := os.ReadFile(history.path)
	if err != nil {
		return nil, false, fmt.Errorf("open the history: %w", err)
	}
	lines := bytes.Split(held, []byte{'\n'})
	isTorn = len(lines[len(lines)-1]) > 0
	for _, line := range lines[:len(lines)-1] {
		var entry Entry
		if err := json.Unmarshal(line, &entry); err != nil {
			return nil, false, fmt.Errorf("read history: %w", err)
		}
		entries = append(entries, entry)
	}
	return entries, isTorn, nil
}
