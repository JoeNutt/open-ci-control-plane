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

var (
	ErrNoRecordOutput             = errors.New("decision recorder has nowhere to write")
	ErrUnidentifiedDecisionRecord = errors.New("decision record carries no decision identifier")
	ErrUnclassifiedOutcome        = errors.New("decision outcome has no class in the HTTP contract")
)

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
	published, err := publishedRecordOf(record)
	if err != nil {
		return err
	}
	if err := json.NewEncoder(recorder.output).Encode(published); err != nil {
		return fmt.Errorf("write the decision record as a JSON line: %w", err)
	}
	return nil
}

type publishedRecord struct {
	DecisionID       string                  `json:"decisionId"`
	BundleRevision   *string                 `json:"bundleRevision,omitempty"`
	OutcomeClass     string                  `json:"outcomeClass"`
	MaskedInput      domain.MaskedInput      `json:"maskedInput"`
	RedactedPointers domain.Declared[string] `json:"redactedPointers"`
}

func publishedRecordOf(record domain.DecisionRecord) (publishedRecord, error) {
	if !record.DecisionID().IsKnown() {
		return publishedRecord{}, ErrUnidentifiedDecisionRecord
	}
	class, err := outcomeClassOf(record.Outcome())
	if err != nil {
		return publishedRecord{}, err
	}
	return publishedRecord{
		DecisionID:       record.DecisionID().String(),
		BundleRevision:   publishedRevisionOf(record.Revision()),
		OutcomeClass:     class,
		MaskedInput:      record.MaskedInput(),
		RedactedPointers: domain.Declared[string](record.MaskedInput().RedactedPointers()),
	}, nil
}

var outcomeClassByOutcome = map[domain.Outcome]string{
	domain.OutcomeDecided:          "decided",
	domain.OutcomeUndefined:        "undecided",
	domain.OutcomeEngineError:      "policy_error",
	domain.OutcomeDeadlineExceeded: "deadline_exceeded",
}

func outcomeClassOf(outcome domain.Outcome) (string, error) {
	class, isClassified := outcomeClassByOutcome[outcome]
	if !isClassified {
		return "", fmt.Errorf("%w: %q", ErrUnclassifiedOutcome, outcome)
	}
	return class, nil
}

func publishedRevisionOf(revision domain.BundleRevision) *string {
	if !revision.IsKnown() {
		return nil
	}
	stated := revision.String()
	return &stated
}
