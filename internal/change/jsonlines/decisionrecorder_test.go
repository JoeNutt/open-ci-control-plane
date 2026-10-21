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

func TestARecorderWithNowhereToWriteIsRefused(t *testing.T) {
	if _, err := jsonlines.ParseDecisionRecorder(nil); !errors.Is(err, jsonlines.ErrNoRecordOutput) {
		t.Fatalf("parse with no output gave %v, want ErrNoRecordOutput", err)
	}
}

func TestARecordedDecisionIsWrittenAsExactlyOneJSONLine(t *testing.T) {
	var written bytes.Buffer
	recorder := jsonlines.NewDecisionRecorder(&written)

	if err := recorder.RecordDecision(context.Background(), recordOf(t)); err != nil {
		t.Fatalf("record: %v", err)
	}

	line, isTerminated := strings.CutSuffix(written.String(), "\n")
	if !isTerminated || strings.Contains(line, "\n") {
		t.Fatalf("wrote %q, want exactly one newline-terminated line", written.String())
	}
	var published struct {
		DecisionID string `json:"decisionId"`
	}
	if err := json.Unmarshal([]byte(line), &published); err != nil {
		t.Fatalf("the line is not one JSON object: %v", err)
	}
	if published.DecisionID != recordedDecisionID {
		t.Errorf("decisionId is %q, want %q", published.DecisionID, recordedDecisionID)
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
	var written bytes.Buffer

	record, err := domain.RecordFor(undecidedStructural(t), document, decisionIDOf(t))
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	if err := jsonlines.NewDecisionRecorder(&written).RecordDecision(context.Background(), record); err != nil {
		t.Fatalf("write: %v", err)
	}

	if strings.Contains(written.String(), syntheticCredential) {
		t.Fatalf("the synthetic credential reached the written line: %s", written.String())
	}
	var published struct {
		MaskedInput      *json.RawMessage `json:"maskedInput"`
		RedactedPointers []string         `json:"redactedPointers"`
	}
	if err := json.Unmarshal(written.Bytes(), &published); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if published.MaskedInput == nil {
		t.Errorf("the line carries no maskedInput, so the absence above proves nothing: %s", written.String())
	}
	if !slices.Contains(published.RedactedPointers, sensitivePointer) {
		t.Errorf("redactedPointers is %q, want it to list %s", published.RedactedPointers, sensitivePointer)
	}
}

func recordOf(t *testing.T) domain.DecisionRecord {
	t.Helper()
	record, err := domain.RecordFor(undecidedStructural(t), domain.Document{}, decisionIDOf(t))
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

func decisionIDOf(t *testing.T) domain.DecisionID {
	t.Helper()
	id, err := domain.ParseDecisionID(recordedDecisionID)
	if err != nil {
		t.Fatalf("parse decision id: %v", err)
	}
	return id
}
