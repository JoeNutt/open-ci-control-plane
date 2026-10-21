// SPDX-License-Identifier: GPL-3.0-or-later

package jsonlines

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var ErrNoRecordOutput = errors.New("decision recorder has nowhere to write")

type DecisionRecorder struct {
	output io.Writer
}

func ParseDecisionRecorder(output io.Writer) (DecisionRecorder, error) {
	if output == nil {
		return DecisionRecorder{}, ErrNoRecordOutput
	}
	return DecisionRecorder{output: output}, nil
}

func (recorder DecisionRecorder) RecordDecision(_ context.Context, record domain.DecisionRecord) error {
	if err := json.NewEncoder(recorder.output).Encode(record); err != nil {
		return fmt.Errorf("write the decision record as a JSON line: %w", err)
	}
	return nil
}
