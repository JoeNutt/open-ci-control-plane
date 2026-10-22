// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "errors"

var ErrHistoryFull = errors.New("the history store is at its bound")

type HistoryLimits struct {
	MaxBytes int64
}

func DeclaredHistoryLimits() HistoryLimits {
	return HistoryLimits{MaxBytes: 256 << 20}
}
