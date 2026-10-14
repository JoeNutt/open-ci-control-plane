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
}

func DeclaredLimits() Limits {
	return Limits{
		MaxChangedFiles:      1000,
		MaxPatchBytesPerFile: 256 << 10,
		MaxTotalPatchBytes:   8 << 20,
		MaxPathBytes:         4096,
		MaxPathSegments:      20,
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
	filePatchBreachIn,
	pathLengthBreachIn,
	pathDepthBreachIn,
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

func pathLengthBreachIn(files []File, limits Limits) error {
	for _, file := range files {
		if len(file.Path) > limits.MaxPathBytes {
			return fmt.Errorf("%w: %q is %d bytes, which exceeds %d",
				ErrPathTooLong, file.Path, len(file.Path), limits.MaxPathBytes)
		}
	}
	return nil
}

func pathDepthBreachIn(files []File, limits Limits) error {
	for _, file := range files {
		segments := strings.Count(file.Path, pathSeparator) + 1
		if segments > limits.MaxPathSegments {
			return fmt.Errorf("%w: %q is %d segments, which exceeds %d",
				ErrPathTooDeep, file.Path, segments, limits.MaxPathSegments)
		}
	}
	return nil
}
