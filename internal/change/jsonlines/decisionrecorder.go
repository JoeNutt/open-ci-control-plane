// SPDX-License-Identifier: GPL-3.0-or-later

package jsonlines

import (
	"context"
	"encoding/json"
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
	return json.NewEncoder(recorder.output).Encode(record)
}
