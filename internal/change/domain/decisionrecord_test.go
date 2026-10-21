// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"encoding/json"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type publishedDecisionRecord struct {
	DecisionID string `json:"decisionId"`
}

func TestADecisionRecordNamesItsDecisionID(t *testing.T) {
	id, err := domain.ParseDecisionID(wellFormedDecisionID)
	if err != nil {
		t.Fatalf("parse decision id: %v", err)
	}
	decided := domain.FamilyDecision{
		Family:   domain.PolicyFamilyStructural,
		Decision: domain.UndecidedDecision(revisionForTest(t)),
	}

	record, err := domain.RecordFor(decided, domain.Document{}, id)
	if err != nil {
		t.Fatalf("record: %v", err)
	}

	encoded, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var published publishedDecisionRecord
	if err := json.Unmarshal(encoded, &published); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if published.DecisionID != wellFormedDecisionID {
		t.Errorf("decisionId is %q, want %q", published.DecisionID, wellFormedDecisionID)
	}
}
