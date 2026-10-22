// SPDX-License-Identifier: GPL-3.0-or-later

package jsonl

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
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
	return &Log{path: path, limits: limits, now: now}, nil
}

func (history *Log) Append(body json.RawMessage) error {
	history.guard.Lock()
	defer history.guard.Unlock()

	line, err := json.Marshal(Entry{At: history.now(), Body: body})
	if err != nil {
		return err
	}
	file, err := os.OpenFile(history.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	held, err := file.Stat()
	if err != nil {
		return errors.Join(err, file.Close())
	}
	if held.Size()+int64(len(line))+1 > history.limits.MaxBytes {
		return errors.Join(domain.ErrHistoryFull, file.Close())
	}
	if _, err := file.Write(append(line, '\n')); err != nil {
		return errors.Join(err, file.Close())
	}
	return errors.Join(file.Sync(), file.Close())
}

func (history *Log) Entries() ([]Entry, error) {
	history.guard.Lock()
	defer history.guard.Unlock()

	file, err := os.Open(history.path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var entries []Entry
	lines := bufio.NewScanner(file)
	for lines.Scan() {
		var entry Entry
		if err := json.Unmarshal(lines.Bytes(), &entry); err != nil {
			return nil, fmt.Errorf("read history: %w", err)
		}
		entries = append(entries, entry)
	}
	return entries, lines.Err()
}
