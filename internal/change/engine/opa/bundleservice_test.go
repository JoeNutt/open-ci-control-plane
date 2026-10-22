// SPDX-License-Identifier: GPL-3.0-or-later

package opa_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/engine/opa"
)

func TestParseBundleServiceRefusesAnythingButAnHTTPLoopbackAddressWithAPort(t *testing.T) {
	for form, stated := range map[string]string{
		"nothing":            "",
		"https":              "https://127.0.0.1:8181",
		"a routable address": "http://10.0.0.1:8181",
		"a host name":        "http://localhost:8181",
		"no port":            "http://127.0.0.1",
		"port zero":          "http://127.0.0.1:0",
		"a path":             "http://127.0.0.1:8181/bundles",
		"a query":            "http://127.0.0.1:8181?x=1",
		"credentials":        "http://user:pass@127.0.0.1:8181",
		"not a URL":          "http://127.0.0.1:port",
	} {
		t.Run(form, func(t *testing.T) {
			if _, err := opa.ParseBundleService(stated); !errors.Is(err, opa.ErrMalformedBundleService) {
				t.Errorf("%q gave %v, want %v", stated, err, opa.ErrMalformedBundleService)
			}
		})
	}
}

func TestParseBundleServiceAcceptsAnIPv4OrIPv6LoopbackAddress(t *testing.T) {
	for _, stated := range []string{"http://127.0.0.1:8181", "http://[::1]:8181"} {
		service, err := opa.ParseBundleService(stated)
		if err != nil || service.String() != stated {
			t.Errorf("%q gave %v, %v; want it back unchanged", stated, service, err)
		}
	}
}
