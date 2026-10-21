// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"encoding/json"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type publishedDecisionRecord struct {
	DecisionID     string `json:"decisionId"`
	BundleRevision string `json:"bundleRevision"`
}

func TestADecisionRecordNamesItsDecisionID(t *testing.T) {
	published := publishedRecordOf(t, undecidedStructuralDecision(t))

	if published.DecisionID != wellFormedDecisionID {
		t.Errorf("decisionId is %q, want %q", published.DecisionID, wellFormedDecisionID)
	}
}

func TestADecisionRecordNamesTheRevisionItsDecisionWasMadeAgainst(t *testing.T) {
	published := publishedRecordOf(t, undecidedStructuralDecision(t))

	if published.BundleRevision != wellFormedRevision {
		t.Errorf("bundleRevision is %q, want %q", published.BundleRevision, wellFormedRevision)
	}
}

func undecidedStructuralDecision(t *testing.T) domain.FamilyDecision {
	t.Helper()
	return domain.FamilyDecision{
		Family:   domain.PolicyFamilyStructural,
		Decision: domain.UndecidedDecision(revisionForTest(t)),
	}
}

func publishedRecordOf(t *testing.T, decided domain.FamilyDecision) publishedDecisionRecord {
	t.Helper()
	id, err := domain.ParseDecisionID(wellFormedDecisionID)
	if err != nil {
		t.Fatalf("parse decision id: %v", err)
	}
	encoded, err := json.Marshal(domain.RecordFor(decided, id))
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var published publishedDecisionRecord
	if err := json.Unmarshal(encoded, &published); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return published
}
