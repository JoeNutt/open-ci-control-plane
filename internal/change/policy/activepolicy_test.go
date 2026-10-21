// SPDX-License-Identifier: GPL-3.0-or-later

package policy_test

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/policy"
)

const (
	firstPolicyCommit  = "1111111111111111111111111111111111111111"
	secondPolicyCommit = "2222222222222222222222222222222222222222"
)

func TestActivePolicyHoldsNoRevisionBeforeOneIsAccepted(t *testing.T) {
	active := policy.NewActivePolicy(slog.New(slog.DiscardHandler))

	if _, err := active.Accepted(); !errors.Is(err, policy.ErrNoPolicyLoaded) {
		t.Fatalf("a fresh active policy gave %v, want ErrNoPolicyLoaded", err)
	}
}

func TestActivePolicyAcceptsAValidCandidateAtItsCommit(t *testing.T) {
	active := activeAtFirstCommit(t)

	if revision := acceptedRevisionOf(t, active); revision != policyCommit(t, firstPolicyCommit) {
		t.Errorf("accepted revision %s, want %s", revision, firstPolicyCommit)
	}
}

func TestActivePolicyServesTheModelParsedAtTheAcceptedCommit(t *testing.T) {
	accepted, err := activeAtFirstCommit(t).Accepted()
	if err != nil {
		t.Fatalf("accepted revision: %v", err)
	}

	bindings := accepted.Model().Bindings()
	if len(bindings) != 1 || bindings[0].Repository().String() != "JoeNutt/open-ci-control-plane" {
		t.Errorf("accepted model binds %v, want the live binding for JoeNutt/open-ci-control-plane", bindings)
	}
}

func TestActivePolicyKeepsThePreviousRevisionWhenACandidateIsRefused(t *testing.T) {
	active := activeAtFirstCommit(t)
	candidate := liveFiles(t)
	candidate[1].Content = []byte(liveBindingPlacingOnePathInTwoGroups(t))

	if err := active.Offer(policyCommit(t, secondPolicyCommit), candidate); !errors.Is(err, domain.ErrPathInTwoGroups) {
		t.Fatalf("a binding placing one path in two groups gave %v, want ErrPathInTwoGroups", err)
	}
	if revision := acceptedRevisionOf(t, active); revision != policyCommit(t, firstPolicyCommit) {
		t.Errorf("accepted revision %s after a refusal, want the previous %s", revision, firstPolicyCommit)
	}
}

func TestActivePolicyLogsARefusalWithItsReasonAndNoFileContent(t *testing.T) {
	var logged bytes.Buffer
	active := policy.NewActivePolicy(slog.New(slog.NewJSONHandler(&logged, nil)))
	const contentMarker = "unlogged_content_marker"
	candidate := liveFiles(t)
	candidate[1].Content = []byte(liveBindingWith(t, "  workflow: open-ci-tests.yml", "  workflow: open-ci-tests.yml\n  "+contentMarker+": 5"))

	if err := active.Offer(policyCommit(t, secondPolicyCommit), candidate); !errors.Is(err, policy.ErrMalformedPolicyFile) {
		t.Fatalf("a binding with an unknown key gave %v, want ErrMalformedPolicyFile", err)
	}
	for _, wanted := range []string{secondPolicyCommit, policy.ErrMalformedPolicyFile.Error()} {
		if !strings.Contains(logged.String(), wanted) {
			t.Errorf("refusal log %q does not name %q", logged.String(), wanted)
		}
	}
	if strings.Contains(logged.String(), contentMarker) {
		t.Errorf("refusal log %q carries file content", logged.String())
	}
}

func activeAtFirstCommit(t *testing.T) *policy.ActivePolicy {
	t.Helper()

	active := policy.NewActivePolicy(slog.New(slog.DiscardHandler))
	if err := active.Offer(policyCommit(t, firstPolicyCommit), liveFiles(t)); err != nil {
		t.Fatalf("offer the live policy files: %v", err)
	}
	return active
}

func acceptedRevisionOf(t *testing.T, active *policy.ActivePolicy) domain.CommitSHA {
	t.Helper()

	accepted, err := active.Accepted()
	if err != nil {
		t.Fatalf("accepted revision: %v", err)
	}
	return accepted.Revision()
}

func policyCommit(t *testing.T, stated string) domain.CommitSHA {
	t.Helper()

	commit, err := domain.ParseCommitSHA(stated)
	if err != nil {
		t.Fatalf("parse commit %s: %v", stated, err)
	}
	return commit
}
