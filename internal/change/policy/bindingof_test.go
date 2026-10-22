// SPDX-License-Identifier: GPL-3.0-or-later

package policy_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/policy"
)

func TestTheActivePolicyServesTheBindingItAcceptedAndRefusesAnyOther(t *testing.T) {
	active := activeAtFirstCommit(t)

	bound, err := active.BindingOf(context.Background(), governedRepositoryOf(t, liveRepository))
	if err != nil {
		t.Fatalf("binding of %s: %v", liveRepository, err)
	}
	if bound.Repository().String() != liveRepository {
		t.Errorf("served the binding for %s, want %s", bound.Repository(), liveRepository)
	}

	_, err = active.BindingOf(context.Background(), governedRepositoryOf(t, "acme/unbound"))
	if !errors.Is(err, domain.ErrNoBinding) {
		t.Errorf("an unbound repository gave %v, want %v", err, domain.ErrNoBinding)
	}

	_, err = activePolicyLoggingTo(t, slog.DiscardHandler).BindingOf(context.Background(), governedRepositoryOf(t, liveRepository))
	if !errors.Is(err, policy.ErrNoPolicyLoaded) {
		t.Errorf("with no policy loaded: %v, want %v", err, policy.ErrNoPolicyLoaded)
	}
}

func governedRepositoryOf(t *testing.T, stated string) domain.GovernedRepository {
	t.Helper()

	repository, err := domain.ParseGovernedRepository(stated)
	if err != nil {
		t.Fatalf("parse repository: %v", err)
	}
	return repository
}
