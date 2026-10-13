// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"slices"
)

type Limits struct {
	MaxChangedFiles      int
	MaxPatchBytesPerFile int
	MaxTotalPatchBytes   int
}

type BoundedFiles struct {
	files []File
}

var ErrTooManyChangedFiles = errors.New("change exceeds the changed file bound")

var ErrFilePatchTooLarge = errors.New("file exceeds the patch byte bound")

var ErrChangePatchTooLarge = errors.New("change exceeds the total patch byte bound")

var boundsOfAChange = []func([]File, Limits) error{
	changedFileCountBreachIn,
	filePatchBreachIn,
	totalPatchBreachIn,
}

func ParseBoundedFiles(files []File, limits Limits) (BoundedFiles, error) {
	for _, breachIn := range boundsOfAChange {
		if breach := breachIn(files, limits); breach != nil {
			return BoundedFiles{}, breach
		}
	}
	return BoundedFiles{files: slices.Clone(files)}, nil
}

func changedFileCountBreachIn(files []File, limits Limits) error {
	if len(files) > limits.MaxChangedFiles {
		return fmt.Errorf("%w: %d exceeds %d",
			ErrTooManyChangedFiles, len(files), limits.MaxChangedFiles)
	}
	return nil
}

func filePatchBreachIn(files []File, limits Limits) error {
	for _, file := range files {
		if file.PatchBytes > limits.MaxPatchBytesPerFile {
			return fmt.Errorf("%w: %q carries %d, which exceeds %d",
				ErrFilePatchTooLarge, file.Path, file.PatchBytes, limits.MaxPatchBytesPerFile)
		}
	}
	return nil
}

func totalPatchBreachIn(files []File, limits Limits) error {
	total := 0
	for _, file := range files {
		total += file.PatchBytes
	}
	if total > limits.MaxTotalPatchBytes {
		return fmt.Errorf("%w: %d exceeds %d",
			ErrChangePatchTooLarge, total, limits.MaxTotalPatchBytes)
	}
	return nil
}
