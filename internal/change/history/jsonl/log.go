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
)

type Entry struct {
	At   time.Time       `json:"at"`
	Body json.RawMessage `json:"body"`
}

type Log struct {
	guard sync.Mutex
	path  string
	now   func() time.Time
}

func OpenLog(path string, now func() time.Time) (*Log, error) {
	return &Log{path: path, now: now}, nil
}

func (log *Log) Append(body json.RawMessage) error {
	log.guard.Lock()
	defer log.guard.Unlock()

	line, err := json.Marshal(Entry{At: log.now(), Body: body})
	if err != nil {
		return err
	}
	file, err := os.OpenFile(log.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(line, '\n')); err != nil {
		return errors.Join(err, file.Close())
	}
	return errors.Join(file.Sync(), file.Close())
}

func (log *Log) Entries() ([]Entry, error) {
	log.guard.Lock()
	defer log.guard.Unlock()

	file, err := os.Open(log.path)
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
