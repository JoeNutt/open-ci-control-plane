// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	sdktest "github.com/open-policy-agent/opa/v1/sdk/test"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const undecidingStructuralPolicy = "package ci.structural\n"

func structuralBundleService(t *testing.T) string {
	t.Helper()
	server := sdktest.MustNewServer(sdktest.MockBundle("/bundles/"+string(domain.PolicyFamilyStructural), map[string]string{
		"policy.rego": undecidingStructuralPolicy,
		".manifest":   `{"revision": "b0b0000000000000000000000000000000000b0b"}`,
	}))
	t.Cleanup(server.Stop)
	return server.URL()
}

func TestAFamilyWhoseBundleNeverArrivesRefusesStartUpByTheDeadlineNamingIt(t *testing.T) {
	deadline := 500 * time.Millisecond
	starting, cancel := context.WithTimeout(context.Background(), deadline)
	defer cancel()
	began := time.Now()

	_, err := startedEnginesFor(starting,
		[]domain.PolicyFamily{domain.PolicyFamilyStructural, domain.PolicyFamilySecurity},
		structuralBundleService(t))

	if !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), string(domain.PolicyFamilySecurity)) {
		t.Errorf("the refusal is not the start-up deadline naming the security family: %v", err)
	}
	if waited := time.Since(began); waited > 4*deadline {
		t.Errorf("start-up waited %s, past its %s deadline", waited, deadline)
	}
}

func TestEnginesStartOnlyForTheConfiguredFamilies(t *testing.T) {
	starting, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	engines, err := startedEnginesFor(starting, []domain.PolicyFamily{domain.PolicyFamilyStructural},
		structuralBundleService(t))
	if err != nil {
		t.Fatalf("start the engines: %v", err)
	}
	t.Cleanup(func() { engines.stop(context.Background()) })

	started := slices.Collect(maps.Keys(engines))
	if !slices.Equal(started, []domain.PolicyFamily{domain.PolicyFamilyStructural}) {
		t.Errorf("engines started for %v, want the structural family only", started)
	}
}
