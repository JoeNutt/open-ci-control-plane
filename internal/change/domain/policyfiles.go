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
	if breach := oversizedPolicyFileIn(files); breach != nil {
		return BoundedPolicyFiles{}, breach
	}
	return BoundedPolicyFiles{files: slices.Clone(files)}, nil
}

func oversizedPolicyFileIn(files []PolicyFile) error {
	oversized := slices.IndexFunc(files, func(file PolicyFile) bool { return len(file.Content) > MaxPolicyFileBytes })
	if oversized < 0 {
		return nil
	}
	return fmt.Errorf("%w: file %d of %d holds %d bytes, exceeding %d",
		ErrPolicyFileTooLarge, oversized+1, len(files), len(files[oversized].Content), MaxPolicyFileBytes)
}
