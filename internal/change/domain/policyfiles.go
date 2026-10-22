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

var ErrMalformedPolicyFilePath = errors.New("policy file path is not a repository-relative path")

type PolicyFile struct {
	Path    string
	Content []byte
}

type BoundedPolicyFiles struct {
	files []PolicyFile
}

var boundsOfPolicyFiles = []func([]PolicyFile) error{
	policyFileCountBreachIn,
	malformedPolicyFilePathIn,
	oversizedPolicyFileIn,
}

func ParseBoundedPolicyFiles(files []PolicyFile) (BoundedPolicyFiles, error) {
	for _, breachIn := range boundsOfPolicyFiles {
		if breach := breachIn(files); breach != nil {
			return BoundedPolicyFiles{}, breach
		}
	}
	return BoundedPolicyFiles{files: slices.Clone(files)}, nil
}

func (bounded BoundedPolicyFiles) Files() []PolicyFile { return slices.Clone(bounded.files) }

func policyFileCountBreachIn(files []PolicyFile) error {
	return policyFileCountBreachOf(len(files))
}

func policyFileCountBreachOf(count int) error {
	if count > MaxPolicyFiles {
		return fmt.Errorf("%w: %d exceeds %d", ErrTooManyPolicyFiles, count, MaxPolicyFiles)
	}
	return nil
}

func malformedPolicyFilePathIn(files []PolicyFile) error {
	malformed := slices.IndexFunc(files, func(file PolicyFile) bool { return !isRepositoryPath(file.Path) })
	if malformed < 0 {
		return nil
	}
	return fmt.Errorf("%w: file %d of %d, %d bytes", ErrMalformedPolicyFilePath, malformed+1, len(files), len(files[malformed].Path))
}

func oversizedPolicyFileIn(files []PolicyFile) error {
	oversized := slices.IndexFunc(files, func(file PolicyFile) bool { return len(file.Content) > MaxPolicyFileBytes })
	if oversized < 0 {
		return nil
	}
	return fmt.Errorf("%w: file %d of %d holds %d bytes, exceeding %d",
		ErrPolicyFileTooLarge, oversized+1, len(files), len(files[oversized].Content), MaxPolicyFileBytes)
}
