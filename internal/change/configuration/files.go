// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

import (
	"errors"
	"fmt"
	"io"
	"os"
)

var ErrOversizedFile = errors.New("the file exceeds its byte bound")

var errEmptyFile = errors.New("the file is empty")

func fileContentOf(stated statedItems, named item, maxBytes int) ([]byte, error) {
	path, err := valueOf[string](stated, named)
	if err != nil {
		return nil, err
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
