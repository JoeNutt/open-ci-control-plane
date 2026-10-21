// SPDX-License-Identifier: GPL-3.0-or-later

package jsonlines_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/jsonlines"
)

const (
	recordedDecisionID = "dec_0123456789abcdef0123456789abcdef"
	recordedRevision   = "c0ffee0000000000000000000000000000c0ffee"
)

var _ domain.DecisionRecorder = jsonlines.DecisionRecorder{}

type publishedRecord struct {
	DecisionID       string           `json:"decisionId"`
	BundleRevision   *string          `json:"bundleRevision"`
	OutcomeClass     string           `json:"outcomeClass"`
	Verdict          string           `json:"verdict"`
	MaskedInput      *json.RawMessage `json:"maskedInput"`
	RedactedPointers json.RawMessage  `json:"redactedPointers"`
}

func TestARecorderWithNowhereToWriteIsRefused(t *testing.T) {
	if _, err := jsonlines.ParseDecisionRecorder(nil); !errors.Is(err, jsonlines.ErrNoRecordOutput) {
		t.Fatalf("parse with no output gave %v, want ErrNoRecordOutput", err)
	}
}

func TestARecordedDecisionIsWrittenAsExactlyOneJSONLine(t *testing.T) {
	written := writtenLineOf(t, recordOf(t, undecidedStructural(t), domain.Document{}))

	line, isTerminated := strings.CutSuffix(written, "\n")
	if !isTerminated || strings.Contains(line, "\n") {
		t.Fatalf("wrote %q, want exactly one newline-terminated line", written)
	}
	if published := publishedFrom(t, line); published.DecisionID != recordedDecisionID {
		t.Errorf("decisionId is %q, want %q", published.DecisionID, recordedDecisionID)
	}
}

func TestARecordNamesTheRevisionItsDecisionWasMadeAgainst(t *testing.T) {
	published := publishedFrom(t, writtenLineOf(t, recordOf(t, undecidedStructural(t), domain.Document{})))

	if published.BundleRevision == nil || *published.BundleRevision != recordedRevision {
		t.Errorf("bundleRevision is %v, want %q", published.BundleRevision, recordedRevision)
	}
}

func TestARecordOfAFaultPublishesNoRevisionRatherThanAnEmptyOne(t *testing.T) {
	published := publishedFrom(t, writtenLineOf(t, recordOf(t, faultedStructural(t), domain.Document{})))

	if published.BundleRevision != nil {
		t.Errorf("a fault record published bundleRevision %q", *published.BundleRevision)
	}
}

func TestARecordOfAnUndecidedDecisionPublishesTheContractsOutcomeClass(t *testing.T) {
	published := publishedFrom(t, writtenLineOf(t, recordOf(t, undecidedStructural(t), domain.Document{})))

	if published.OutcomeClass != "undecided" {
		t.Errorf("outcomeClass is %q, want %q", published.OutcomeClass, "undecided")
	}
}

func TestARecordOfAnUndecidedDecisionThatFailsClosedPublishesARefusal(t *testing.T) {
	published := publishedFrom(t, writtenLineOf(t, recordOf(t, undecidedStructural(t), domain.Document{})))

	if published.Verdict != "refuse" {
		t.Errorf("verdict is %q, want %q", published.Verdict, "refuse")
	}
}

func TestARecordWithNothingMaskedListsNoRedactionsAsAnEmptyArray(t *testing.T) {
	published := publishedFrom(t, writtenLineOf(t, recordOf(t, undecidedStructural(t), domain.Document{})))

	if string(published.RedactedPointers) != "[]" {
		t.Errorf("redactedPointers is %s, want []", published.RedactedPointers)
	}
}

func TestARecordWithNoDecisionIDIsRefusedAndNothingIsWritten(t *testing.T) {
	var written bytes.Buffer

	err := recorderWritingTo(t, &written).RecordDecision(context.Background(), domain.DecisionRecord{})

	if !errors.Is(err, jsonlines.ErrUnidentifiedDecisionRecord) {
		t.Fatalf("recorded with %v, want ErrUnidentifiedDecisionRecord", err)
	}
	if written.Len() != 0 {
		t.Errorf("an unidentified record still wrote %q", written.String())
	}
}

func TestASensitiveValueReachesNoWrittenLineAndItsPointerIsListedAsRedacted(t *testing.T) {
	const syntheticCredential = "ghp_SyntheticCredentialForSEC8Only0000"
	const sensitivePointer = "/analyzers/findings/0"
	document := domain.Document{}
	document.Analyzers.Findings = domain.Declared[domain.Finding]{
		{Analyzer: "gitleaks", RuleID: "github-pat", Path: "config/" + syntheticCredential},
	}
	document.Diff.Sensitive = domain.Declared[string]{sensitivePointer}

	written := writtenLineOf(t, recordOf(t, undecidedStructural(t), document))

	if strings.Contains(written, syntheticCredential) {
		t.Fatalf("the synthetic credential reached the written line: %s", written)
	}
	published := publishedFrom(t, written)
	if published.MaskedInput == nil {
		t.Errorf("the line carries no maskedInput, so the absence above proves nothing: %s", written)
	}
	var redacted []string
	if err := json.Unmarshal(published.RedactedPointers, &redacted); err != nil {
		t.Fatalf("decode redactedPointers: %v", err)
	}
	if !slices.Contains(redacted, sensitivePointer) {
		t.Errorf("redactedPointers is %q, want it to list %s", redacted, sensitivePointer)
	}
}

func writtenLineOf(t *testing.T, record domain.DecisionRecord) string {
	t.Helper()
	var written bytes.Buffer
	if err := recorderWritingTo(t, &written).RecordDecision(context.Background(), record); err != nil {
		t.Fatalf("record: %v", err)
	}
	return written.String()
}

func publishedFrom(t *testing.T, line string) publishedRecord {
	t.Helper()
	var published publishedRecord
	if err := json.Unmarshal([]byte(line), &published); err != nil {
		t.Fatalf("the line is not one JSON object: %v", err)
	}
	return published
}

func recorderWritingTo(t *testing.T, output *bytes.Buffer) jsonlines.DecisionRecorder {
	t.Helper()
	recorder, err := jsonlines.ParseDecisionRecorder(output)
	if err != nil {
		t.Fatalf("parse recorder: %v", err)
	}
	return recorder
}

func recordOf(t *testing.T, decided domain.FamilyDecision, document domain.Document) domain.DecisionRecord {
	t.Helper()
	id, err := domain.ParseDecisionID(recordedDecisionID)
	if err != nil {
		t.Fatalf("parse decision id: %v", err)
	}
	record, err := domain.RecordFor(decided, document, id)
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	return record
}

func undecidedStructural(t *testing.T) domain.FamilyDecision {
	t.Helper()
	revision, err := domain.ParseBundleRevision(recordedRevision)
	if err != nil {
		t.Fatalf("parse revision: %v", err)
	}
	return domain.FamilyDecision{
		Family:   domain.PolicyFamilyStructural,
		Decision: domain.UndecidedDecision(revision),
	}
}

func faultedStructural(t *testing.T) domain.FamilyDecision {
	t.Helper()
	engines := map[domain.PolicyFamily]domain.FamilyEngine{}
	for _, family := range domain.DeclaredPolicyFamilies() {
		engines[family] = faultingEngine{}
	}
	gate, err := domain.ParseGate(engines)
	if err != nil {
		t.Fatalf("parse gate: %v", err)
	}
	return gate.DecisionsFor(context.Background(), domain.Document{})[0]
}

type faultingEngine struct{}

func (faultingEngine) Decide(context.Context, domain.Document) (domain.Decision, error) {
	return domain.Decision{}, errors.New("conflicting rule outputs")
}
