// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/configuration"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestAPlanTheEvaluationRecordsIsFoundAfterARestart(t *testing.T) {
	historyFile := filepath.Join(t.TempDir(), "history.jsonl")
	configured := configurationWith(t, func(stated map[string]json.RawMessage) {
		stated["historyFile"] = quoted(t, historyFile)
	})
	dispatched := ledgeredPlanOf(t)

	if err := evaluationOf(t, configured).ledger.RecordDispatch(context.Background(), dispatched); err != nil {
		t.Fatalf("record dispatch: %v", err)
	}

	found, err := evaluationOf(t, configured).ledger.DispatchedPlanFor(
		context.Background(), dispatched.Locator.GovernedRepository(), dispatched.Head)
	if err != nil {
		t.Fatalf("after a restart the evaluation's ledger does not hold the plan: %v", err)
	}
	if found.Locator != dispatched.Locator || found.DecisionID != dispatched.DecisionID {
		t.Errorf("after a restart the ledger holds %+v, want %+v", found, dispatched)
	}
}

func evaluationOf(t *testing.T, configured configuration.Configuration) evaluating {
	t.Helper()

	endpoint := func() (forgeEndpoint, error) {
		return forgeEndpoint{baseURL: forgeBaseURL, trusted: x509.NewCertPool()}, nil
	}
	evaluation, err := evaluatingOf(configured, endpoint, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("build the evaluation: %v", err)
	}
	return evaluation
}

func ledgeredPlanOf(t *testing.T) domain.DispatchedPlan {
	t.Helper()

	head, err := domain.ParseCommitSHA("9f1cabc0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6")
	if err != nil {
		t.Fatalf("parse head: %v", err)
	}
	id, err := domain.ParseDecisionID("dec_0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("parse decision ID: %v", err)
	}
	return domain.DispatchedPlan{Locator: recordedPullRequest(t), Head: head, DecisionID: id}
}
