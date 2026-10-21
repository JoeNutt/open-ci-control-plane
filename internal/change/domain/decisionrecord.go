// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"encoding/json"
	"errors"
)

var ErrUnidentifiedDecisionRecord = errors.New("decision record carries no decision identifier")

type DecisionRecord struct {
	decisionID DecisionID
	revision   BundleRevision
	masked     MaskedInput
}

func RecordFor(decided FamilyDecision, document Document, id DecisionID) (DecisionRecord, error) {
	masked, err := MaskedInputOf(document)
	if err != nil {
		return DecisionRecord{}, err
	}
	return DecisionRecord{decisionID: id, revision: decided.Decision.Revision(), masked: masked}, nil
}

func (record DecisionRecord) MarshalJSON() ([]byte, error) {
	if !record.decisionID.isKnown() {
		return nil, ErrUnidentifiedDecisionRecord
	}
	return json.Marshal(publishedRecord{
		DecisionID:       record.decisionID.String(),
		BundleRevision:   publishedRevisionOf(record.revision),
		OutcomeClass:     "undecided",
		MaskedInput:      record.masked,
		RedactedPointers: Declared[string](record.masked.RedactedPointers()),
	})
}

type publishedRecord struct {
	DecisionID       string           `json:"decisionId"`
	BundleRevision   *string          `json:"bundleRevision,omitempty"`
	OutcomeClass     string           `json:"outcomeClass"`
	MaskedInput      MaskedInput      `json:"maskedInput"`
	RedactedPointers Declared[string] `json:"redactedPointers"`
}

func publishedRevisionOf(revision BundleRevision) *string {
	if !revision.isKnown() {
		return nil
	}
	stated := revision.String()
	return &stated
}
