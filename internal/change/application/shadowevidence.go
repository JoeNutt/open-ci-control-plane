// SPDX-License-Identifier: GPL-3.0-or-later

package application

import (
	"context"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type shadowEvidence struct {
	runs    []domain.ShadowedRun
	escapes []domain.Escape
}

func shadowEvidenceIn(ctx context.Context, runs domain.ShadowRuns, escapes domain.EscapeHistory) (shadowEvidence, error) {
	shadowedRuns, err := runs.ShadowedRuns(ctx)
	if err != nil {
		return shadowEvidence{}, fmt.Errorf("read the shadow runs: %w", err)
	}
	recordedEscapes, err := escapes.RecordedEscapes(ctx)
	if err != nil {
		return shadowEvidence{}, fmt.Errorf("read the escapes: %w", err)
	}
	return shadowEvidence{runs: shadowedRuns, escapes: recordedEscapes}, nil
}
