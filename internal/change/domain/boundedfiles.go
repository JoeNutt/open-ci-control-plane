// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

const pathSeparator = "/"

type Limits struct {
	MaxChangedFiles      int
	MaxPatchBytesPerFile int
	MaxTotalPatchBytes   int
	MaxPathBytes         int
	MaxPathSegments      int
	MaxSourceBytes       int
	MaxImportsPerFile    int
	MaxImportsPerChange  int
	MaxGraphEdges        int
	MaxStructuralBytes   int
}

func DeclaredLimits() Limits {
	return Limits{
		MaxChangedFiles:      1000,
		MaxPatchBytesPerFile: 256 << 10,
		MaxTotalPatchBytes:   8 << 20,
		MaxPathBytes:         4096,
		MaxPathSegments:      20,
		MaxSourceBytes:       1 << 20,
		MaxImportsPerFile:    2000,
		MaxImportsPerChange:  50000,
		MaxGraphEdges:        20000,
		MaxStructuralBytes:   4 << 20,
	}
}

type BoundedFiles struct {
	files []File
}

var ErrTooManyChangedFiles = errors.New("change exceeds the changed file bound")

var ErrFilePatchTooLarge = errors.New("file exceeds the patch byte bound")

var ErrChangePatchTooLarge = errors.New("change exceeds the total patch byte bound")

var ErrPathTooLong = errors.New("path exceeds the path byte bound")

var ErrPathTooDeep = errors.New("path exceeds the path segment bound")

var boundsOfAChange = []func([]File, Limits) error{
	changedFileCountBreachIn,
	perFileBreachIn,
	totalPatchBreachIn,
}

var perFileBounds = []struct {
	measurementOf func(File) int
	allowanceIn   func(Limits) int
	breach        error
}{
	{patchBytesOf, func(limits Limits) int { return limits.MaxPatchBytesPerFile }, ErrFilePatchTooLarge},
	{pathBytesOf, func(limits Limits) int { return limits.MaxPathBytes }, ErrPathTooLong},
	{pathSegmentsOf, func(limits Limits) int { return limits.MaxPathSegments }, ErrPathTooDeep},
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

func perFileBreachIn(files []File, limits Limits) error {
	for _, bound := range perFileBounds {
		allowed := bound.allowanceIn(limits)
		breaching := slices.IndexFunc(files, func(file File) bool {
			return bound.measurementOf(file) > allowed
		})
		if breaching >= 0 {
			offending := files[breaching]
			return fmt.Errorf("%w: %q measures %d, which exceeds %d",
				bound.breach, offending.Path, bound.measurementOf(offending), allowed)
		}
	}
	return nil
}

func patchBytesOf(file File) int { return file.PatchBytes }

func pathBytesOf(file File) int { return len(file.Path) }

func pathSegmentsOf(file File) int { return strings.Count(file.Path, pathSeparator) + 1 }
