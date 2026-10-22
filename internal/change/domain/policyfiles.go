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

var ErrMalformedPolicyFileSize = errors.New("policy file is listed with a negative size")

var ErrMalformedPolicyFilePath = errors.New("policy file path is not a repository-relative path")

type PolicyFile struct {
	Path    string
	Content []byte
}

type BoundedPolicyFiles struct {
	files []PolicyFile
}

var boundsOfPolicyFiles = []func([]ListedFile) error{
	policyFileCountBreachIn,
	malformedPolicyFilePathIn,
	negativePolicyFileSizeIn,
	oversizedPolicyFileIn,
}

func ParseBoundedPolicyFiles(files []PolicyFile) (BoundedPolicyFiles, error) {
	if breach := policyFileBoundBreachIn(listedPolicyFilesOf(files)); breach != nil {
		return BoundedPolicyFiles{}, breach
	}
	return BoundedPolicyFiles{files: slices.Clone(files)}, nil
}

func (bounded BoundedPolicyFiles) Files() []PolicyFile { return slices.Clone(bounded.files) }

func policyFileBoundBreachIn(listed []ListedFile) error {
	for _, breachIn := range boundsOfPolicyFiles {
		if breach := breachIn(listed); breach != nil {
			return breach
		}
	}
	return nil
}

func listedPolicyFilesOf(files []PolicyFile) []ListedFile {
	listed := make([]ListedFile, 0, len(files))
	for _, file := range files {
		listed = append(listed, ListedFile{Path: file.Path, Bytes: len(file.Content)})
	}
	return listed
}

func policyFileCountBreachIn(listed []ListedFile) error {
	if len(listed) > MaxPolicyFiles {
		return fmt.Errorf("%w: %d exceeds %d", ErrTooManyPolicyFiles, len(listed), MaxPolicyFiles)
	}
	return nil
}

func malformedPolicyFilePathIn(listed []ListedFile) error {
	malformed := slices.IndexFunc(listed, func(file ListedFile) bool { return !isRepositoryPath(file.Path) })
	if malformed < 0 {
		return nil
	}
	return fmt.Errorf("%w: file %d of %d, %d bytes", ErrMalformedPolicyFilePath, malformed+1, len(listed), len(listed[malformed].Path))
}

func oversizedPolicyFileIn(listed []ListedFile) error {
	oversized := slices.IndexFunc(listed, func(file ListedFile) bool { return file.Bytes > MaxPolicyFileBytes })
	if oversized < 0 {
		return nil
	}
	return fmt.Errorf("%w: file %d of %d holds %d bytes, exceeding %d",
		ErrPolicyFileTooLarge, oversized+1, len(listed), listed[oversized].Bytes, MaxPolicyFileBytes)
}

func negativePolicyFileSizeIn(listed []ListedFile) error {
	negative := slices.IndexFunc(listed, func(file ListedFile) bool { return file.Bytes < 0 })
	if negative < 0 {
		return nil
	}
	return fmt.Errorf("%w: file %d of %d", ErrMalformedPolicyFileSize, negative+1, len(listed))
}
