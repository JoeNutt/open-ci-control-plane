// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"encoding/json"
	"errors"
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

func TestARecordOfAFaultPublishesNoRevisionRatherThanAnEmptyOne(t *testing.T) {
	faulted := domain.FamilyDecision{
		Family:   domain.PolicyFamilyStructural,
		Decision: evaluatedWith(t, &statedEngine{fault: errors.New("conflicting rule outputs")}),
	}
	id, err := domain.ParseDecisionID(wellFormedDecisionID)
	if err != nil {
		t.Fatalf("parse decision id: %v", err)
	}

	encoded, err := json.Marshal(domain.RecordFor(faulted, id))
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var published struct {
		BundleRevision *string `json:"bundleRevision"`
	}
	if err := json.Unmarshal(encoded, &published); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if published.BundleRevision != nil {
		t.Errorf("a fault record published bundleRevision %q: %s", *published.BundleRevision, encoded)
	}
}

func TestARecordWithNoDecisionIDRefusesToEncode(t *testing.T) {
	encoded, err := json.Marshal(domain.DecisionRecord{})

	if !errors.Is(err, domain.ErrUnidentifiedDecisionRecord) {
		t.Fatalf("encoded %s with %v, want ErrUnidentifiedDecisionRecord", encoded, err)
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
	var published publishedDecisionRecord
	if err := json.Unmarshal(encodedRecordOf(t, decided), &published); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return published
}

func encodedRecordOf(t *testing.T, decided domain.FamilyDecision) []byte {
	t.Helper()
	id, err := domain.ParseDecisionID(wellFormedDecisionID)
	if err != nil {
		t.Fatalf("parse decision id: %v", err)
	}
	encoded, err := json.Marshal(domain.RecordFor(decided, id))
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	return encoded
}
