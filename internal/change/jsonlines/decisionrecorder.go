// SPDX-License-Identifier: GPL-3.0-or-later

package jsonlines

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type DecisionRecorder struct {
	output io.Writer
}

func NewDecisionRecorder(output io.Writer) DecisionRecorder {
	return DecisionRecorder{output: output}
}

func (recorder DecisionRecorder) RecordDecision(_ context.Context, record domain.DecisionRecord) error {
	if err := json.NewEncoder(recorder.output).Encode(record); err != nil {
		return fmt.Errorf("write the decision record as a JSON line: %w", err)
	}
	return nil
}
