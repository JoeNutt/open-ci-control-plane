// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
)

var ErrCoverageBoundExceeded = errors.New("coverage artifact exceeds a bound")

var ErrCompressedArtifactTooLarge = fmt.Errorf("%w: compressed bytes", ErrCoverageBoundExceeded)

type CoverageLimits struct {
	MaxCompressedBytes int64
}

func DeclaredCoverageLimits() CoverageLimits {
	return CoverageLimits{MaxCompressedBytes: 64 << 20}
}

func (limits CoverageLimits) DeclaredBytesBreach(declaredBytes int64) error {
	if declaredBytes > limits.MaxCompressedBytes {
		return fmt.Errorf("%w: %d declared exceeds %d",
			ErrCompressedArtifactTooLarge, declaredBytes, limits.MaxCompressedBytes)
	}
	return nil
}
