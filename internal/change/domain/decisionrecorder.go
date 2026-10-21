// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "context"

type DecisionRecorder interface {
	RecordDecision(ctx context.Context, record DecisionRecord) error
}
