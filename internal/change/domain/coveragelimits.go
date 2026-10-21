// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "errors"

var ErrCoverageBoundExceeded = errors.New("coverage artifact exceeds a bound")

type CoverageLimits struct {
	MaxCompressedBytes int64
}

func DeclaredCoverageLimits() CoverageLimits {
	return CoverageLimits{MaxCompressedBytes: 64 << 20}
}
