// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"
	"maps"
	"slices"
	"testing"
	"time"

	sdktest "github.com/open-policy-agent/opa/v1/sdk/test"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const undecidingStructuralPolicy = "package ci.structural\n"

func bundleServiceFor(t *testing.T, family domain.PolicyFamily) string {
	t.Helper()
	server := sdktest.MustNewServer(sdktest.MockBundle("/bundles/"+string(family), map[string]string{
		"policy.rego": undecidingStructuralPolicy,
		".manifest":   `{"revision": "b0b0000000000000000000000000000000000b0b"}`,
	}))
	t.Cleanup(server.Stop)
	return server.URL()
}

func TestEnginesStartOnlyForTheConfiguredFamilies(t *testing.T) {
	starting, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	engines, err := startedEnginesFor(starting, []domain.PolicyFamily{domain.PolicyFamilyStructural},
		bundleServiceFor(t, domain.PolicyFamilyStructural))
	if err != nil {
		t.Fatalf("start the engines: %v", err)
	}
	t.Cleanup(func() { engines.stop(context.Background()) })

	started := slices.Collect(maps.Keys(engines))
	if !slices.Equal(started, []domain.PolicyFamily{domain.PolicyFamilyStructural}) {
		t.Errorf("engines started for %v, want the structural family only", started)
	}
}
