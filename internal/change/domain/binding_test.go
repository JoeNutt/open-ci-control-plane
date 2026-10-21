// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseBindingKeepsWhatItStates(t *testing.T) {
	binding, err := domain.ParseBinding(thisRepositorysBinding())
	if err != nil {
		t.Fatalf("parse binding: %v", err)
	}
	if binding.Repository().String() != "JoeNutt/open-ci-control-plane" {
		t.Errorf("repository is %q", binding.Repository())
	}
	if binding.Workflow().String() != "open-ci-tests.yml" {
		t.Errorf("workflow is %q", binding.Workflow())
	}
}

func TestParseBindingRefusesABindingItCannotTrust(t *testing.T) {
	refused := map[string]struct {
		stated func(*domain.BindingStatement)
		reason error
	}{
		"a malformed repository": {
			stated: func(s *domain.BindingStatement) { s.Repository = "open-ci-control-plane" },
			reason: domain.ErrMalformedRepository,
		},
		"a missing workflow": {
			stated: func(s *domain.BindingStatement) { s.Workflow = "" },
			reason: domain.ErrMissingWorkflow,
		},
	}
	for form, refusal := range refused {
		t.Run(form, func(t *testing.T) {
			stated := thisRepositorysBinding()
			refusal.stated(&stated)
			if _, err := domain.ParseBinding(stated); !errors.Is(err, refusal.reason) {
				t.Fatalf("%s gave %v, want %v", form, err, refusal.reason)
			}
		})
	}
}

func thisRepositorysBinding() domain.BindingStatement {
	return domain.BindingStatement{
		Repository: "JoeNutt/open-ci-control-plane",
		Workflow:   "open-ci-tests.yml",
	}
}
