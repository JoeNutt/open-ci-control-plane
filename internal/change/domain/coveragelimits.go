// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
)

var ErrCoverageBoundExceeded = errors.New("coverage artifact exceeds a bound")

var ErrCompressedArtifactTooLarge = fmt.Errorf("%w: compressed bytes", ErrCoverageBoundExceeded)

var ErrDecompressedArtifactTooLarge = fmt.Errorf("%w: decompressed bytes", ErrCoverageBoundExceeded)

var ErrTooManyFileEntries = fmt.Errorf("%w: file entries", ErrCoverageBoundExceeded)

var ErrTooManySuites = fmt.Errorf("%w: suites per repository", ErrCoverageBoundExceeded)

type CoverageLimits struct {
	MaxCompressedBytes     int64
	MaxDecompressedBytes   int64
	MaxFileEntries         int
	MaxSuitesPerRepository int
	MaxUploadsPerMinute    int
}

func DeclaredCoverageLimits() CoverageLimits {
	return CoverageLimits{
		MaxCompressedBytes:     64 << 20,
		MaxDecompressedBytes:   256 << 20,
		MaxFileEntries:         200_000,
		MaxSuitesPerRepository: 5_000,
		MaxUploadsPerMinute:    60,
	}
}

func (limits CoverageLimits) FileEntriesBreach(fileEntries int) error {
	return breachOf(ErrTooManyFileEntries, int64(fileEntries), int64(limits.MaxFileEntries))
}

func (limits CoverageLimits) SuitesBreach(suites int) error {
	return breachOf(ErrTooManySuites, int64(suites), int64(limits.MaxSuitesPerRepository))
}

func (limits CoverageLimits) CompressedBytesBreach(compressedBytes int64) error {
	return breachOf(ErrCompressedArtifactTooLarge, compressedBytes, limits.MaxCompressedBytes)
}

func (limits CoverageLimits) DecompressedBytesBreach(decompressedBytes int64) error {
	return breachOf(ErrDecompressedArtifactTooLarge, decompressedBytes, limits.MaxDecompressedBytes)
}

func breachOf(bound error, measured, allowed int64) error {
	if measured > allowed {
		return fmt.Errorf("%w: %d exceeds %d", bound, measured, allowed)
	}
	return nil
}
