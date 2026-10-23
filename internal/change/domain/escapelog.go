// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "context"

type EscapeLog interface {
	RecordEscape(ctx context.Context, escape Escape) error
}
