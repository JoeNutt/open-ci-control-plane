// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseHTTPSOriginRefusesAnythingButAnHTTPSOrigin(t *testing.T) {
	for form, stated := range map[string]string{
		"nothing":           "",
		"no scheme":         "api.github.com",
		"plain http":        "http://api.github.com",
		"no host":           "https://",
		"a trailing slash":  "https://api.github.com/",
		"a path":            "https://api.github.com/api/v3",
		"a query":           "https://api.github.com?per_page=1",
		"an empty query":    "https://api.github.com?",
		"a fragment":        "https://api.github.com#top",
		"credentials":       "https://user:pass@api.github.com",
		"an unparsable URL": "https://api.github.com:port",
	} {
		t.Run(form, func(t *testing.T) {
			if _, err := domain.ParseHTTPSOrigin(stated); !errors.Is(err, domain.ErrMalformedHTTPSOrigin) {
				t.Errorf("%q gave %v, want %v", stated, err, domain.ErrMalformedHTTPSOrigin)
			}
		})
	}
}

func TestAnHTTPSOriginIsTheAddressStated(t *testing.T) {
	const stated = "https://api.github.com"

	origin, err := domain.ParseHTTPSOrigin(stated)
	if err != nil {
		t.Fatalf("parse %q: %v", stated, err)
	}

	if origin.String() != stated {
		t.Errorf("the origin is %q, want %q", origin.String(), stated)
	}
}

func TestTheRefusalOfAnOriginCarryingCredentialsDoesNotRepeatThem(t *testing.T) {
	const password = "s3cret-in-the-url"

	for _, stated := range []string{
		"https://user:" + password + "@api.github.com",
		"https://user:" + password + "@api.github.com:port",
	} {
		_, err := domain.ParseHTTPSOrigin(stated)

		if err == nil || strings.Contains(err.Error(), password) {
			t.Errorf("the refusal is %v, want one that does not carry the password", err)
		}
	}
}
