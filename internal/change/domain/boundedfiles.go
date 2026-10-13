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

func ParseBoundedFiles(files []File, limits Limits) (BoundedFiles, error) {
	if len(files) > limits.MaxChangedFiles {
		return BoundedFiles{}, fmt.Errorf("%w: %d exceeds %d",
			ErrTooManyChangedFiles, len(files), limits.MaxChangedFiles)
	}
	totalPatchBytes := 0
	for _, file := range files {
		if file.PatchBytes > limits.MaxPatchBytesPerFile {
			return BoundedFiles{}, fmt.Errorf("%w: %q carries %d, which exceeds %d",
				ErrFilePatchTooLarge, file.Path, file.PatchBytes, limits.MaxPatchBytesPerFile)
		}
		totalPatchBytes += file.PatchBytes
	}
	if totalPatchBytes > limits.MaxTotalPatchBytes {
		return BoundedFiles{}, fmt.Errorf("%w: %d exceeds %d",
			ErrChangePatchTooLarge, totalPatchBytes, limits.MaxTotalPatchBytes)
	}
	return BoundedFiles{files: slices.Clone(files)}, nil
}
