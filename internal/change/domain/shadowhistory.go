// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"context"
	"errors"
)

var ErrNoShadowHistory = errors.New("no shadow history is configured")

type ShadowHistory interface {
	VerifyReadable(ctx context.Context) error
}
