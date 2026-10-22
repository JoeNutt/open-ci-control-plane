// SPDX-License-Identifier: GPL-3.0-or-later

package jsonl

import (
	"bufio"
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
	entries, err := history.Entries()
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	cutoff := history.now().Add(-history.limits.Retention)
	held := len(entries)
	entries = slices.DeleteFunc(entries, func(entry Entry) bool { return entry.At.Before(cutoff) })
	if len(entries) == held {
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

func (history *Log) Entries() (entries []Entry, err error) {
	history.guard.Lock()
	defer history.guard.Unlock()

	file, err := os.Open(history.path)
	if err != nil {
		return nil, fmt.Errorf("open the history: %w", err)
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	lines := bufio.NewScanner(file)
	lines.Buffer(nil, int(history.limits.MaxBytes))
	for lines.Scan() {
		var entry Entry
		if err := json.Unmarshal(lines.Bytes(), &entry); err != nil {
			return nil, fmt.Errorf("read history: %w", err)
		}
		entries = append(entries, entry)
	}
	return entries, lines.Err()
}
