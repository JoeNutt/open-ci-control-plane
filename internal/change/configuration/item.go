// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
)

var ErrMissingItem = errors.New("the configuration is missing a required item")

var ErrInvalidItem = errors.New("a configuration item is invalid")

var ErrOversizedFile = errors.New("the file exceeds its byte bound")

var errEmptyFile = errors.New("the file is empty")

var errNonPositiveID = errors.New("a GitHub ID is not positive")

const firstGitHubID = 1

type item string

func (absent item) missing() error {
	return fmt.Errorf("%w: %s", ErrMissingItem, absent)
}

func (refused item) invalid(reason error) error {
	return fmt.Errorf("%w: %s: %w", ErrInvalidItem, refused, reason)
}

func (named item) contentAt(path string, maxBytes int) ([]byte, error) {
	if path == "" {
		return nil, named.missing()
	}
	content, err := boundedContentOf(path, maxBytes)
	if err != nil {
		return nil, named.invalid(err)
	}
	if len(content) == 0 {
		return nil, named.invalid(errEmptyFile)
	}
	return content, nil
}

func repeatsAnEntry[Entry cmp.Ordered](stated []Entry) bool {
	return len(slices.Compact(slices.Sorted(slices.Values(stated)))) != len(stated)
}

func boundedContentOf(path string, maxBytes int) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	content, readErr := io.ReadAll(io.LimitReader(file, int64(maxBytes)+1))
	if err := errors.Join(readErr, file.Close()); err != nil {
		return nil, err
	}
	if len(content) > maxBytes {
		return nil, fmt.Errorf("%w: more than %d bytes", ErrOversizedFile, maxBytes)
	}
	return content, nil
}
