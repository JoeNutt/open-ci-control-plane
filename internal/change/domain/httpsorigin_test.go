// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseHTTPSOriginReadsAsTheOriginItWasGiven(t *testing.T) {
	const stated = "https://control-plane.example:8443"

	origin, err := domain.ParseHTTPSOrigin(stated)

	if err != nil || origin.String() != stated {
		t.Fatalf("parsed %q (%v), want %q", origin, err, stated)
	}
}

func TestParseHTTPSOriginRefusesAnythingButAnHTTPSOrigin(t *testing.T) {
	for _, stated := range []string{
		"", "http://control-plane.example", "control-plane.example", "https://", "https://%zz",
		"https://operator@control-plane.example", "https://control-plane.example/",
		"https://control-plane.example/ci", "https://control-plane.example?audience=1",
		"https://control-plane.example?", "https://control-plane.example#top",
	} {
		if _, err := domain.ParseHTTPSOrigin(stated); !errors.Is(err, domain.ErrMalformedHTTPSOrigin) {
			t.Errorf("%q: error is %v, want %v", stated, err, domain.ErrMalformedHTTPSOrigin)
		}
	}
}
