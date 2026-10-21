// SPDX-License-Identifier: GPL-3.0-or-later

package jsonlines_test

import (
	"bytes"
	"context"
	"encoding/json"
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

func recordOf(t *testing.T) domain.DecisionRecord {
	t.Helper()
	revision, err := domain.ParseBundleRevision(recordedRevision)
	if err != nil {
		t.Fatalf("parse revision: %v", err)
	}
	id, err := domain.ParseDecisionID(recordedDecisionID)
	if err != nil {
		t.Fatalf("parse decision id: %v", err)
	}
	decided := domain.FamilyDecision{
		Family:   domain.PolicyFamilyStructural,
		Decision: domain.UndecidedDecision(revision),
	}
	return domain.RecordFor(decided, id)
}
