// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "context"

type ShadowRuns interface {
	ShadowedRuns(ctx context.Context) ([]ShadowedRun, error)
}

type EscapeHistory interface {
	RecordedEscapes(ctx context.Context) ([]Escape, error)
}
