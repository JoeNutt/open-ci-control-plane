// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"time"
)

var ErrHistoryFull = errors.New("the history store is at its bound")

type HistoryLimits struct {
	MaxBytes  int64
	Retention time.Duration
}

func DeclaredHistoryLimits() HistoryLimits {
	return HistoryLimits{MaxBytes: 256 << 20, Retention: 90 * 24 * time.Hour}
}
