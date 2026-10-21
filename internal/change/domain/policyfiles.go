// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"slices"
)

const (
	MaxPolicyFiles     = 256
	MaxPolicyFileBytes = 1 << 20
)

var ErrTooManyPolicyFiles = errors.New("policy repository exceeds the policy file bound")

var ErrPolicyFileTooLarge = errors.New("policy file exceeds the policy file byte bound")

type PolicyFile struct {
	Path    string
	Content []byte
}

type BoundedPolicyFiles struct {
	files []PolicyFile
}

func ParseBoundedPolicyFiles(files []PolicyFile) (BoundedPolicyFiles, error) {
	if len(files) > MaxPolicyFiles {
		return BoundedPolicyFiles{}, fmt.Errorf("%w: %d exceeds %d", ErrTooManyPolicyFiles, len(files), MaxPolicyFiles)
	}
	for _, file := range files {
		if len(file.Content) > MaxPolicyFileBytes {
			return BoundedPolicyFiles{}, ErrPolicyFileTooLarge
		}
	}
	return BoundedPolicyFiles{files: slices.Clone(files)}, nil
}
