// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
)

var ErrCoverageBoundExceeded = errors.New("coverage artifact exceeds a bound")

var ErrCompressedArtifactTooLarge = fmt.Errorf("%w: compressed bytes", ErrCoverageBoundExceeded)

var ErrDecompressedArtifactTooLarge = fmt.Errorf("%w: decompressed bytes", ErrCoverageBoundExceeded)

type CoverageLimits struct {
	MaxCompressedBytes   int64
	MaxDecompressedBytes int64
}

func DeclaredCoverageLimits() CoverageLimits {
	return CoverageLimits{MaxCompressedBytes: 64 << 20, MaxDecompressedBytes: 256 << 20}
}

func (limits CoverageLimits) CompressedBytesBreach(compressedBytes int64) error {
	if compressedBytes > limits.MaxCompressedBytes {
		return fmt.Errorf("%w: %d exceeds %d",
			ErrCompressedArtifactTooLarge, compressedBytes, limits.MaxCompressedBytes)
	}
	return nil
}

func (limits CoverageLimits) DecompressedBytesBreach(decompressedBytes int64) error {
	if decompressedBytes > limits.MaxDecompressedBytes {
		return fmt.Errorf("%w: %d exceeds %d",
			ErrDecompressedArtifactTooLarge, decompressedBytes, limits.MaxDecompressedBytes)
	}
	return nil
}
