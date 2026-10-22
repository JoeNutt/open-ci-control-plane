// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"net/url"
	"slices"
)

const httpsScheme = "https"

var ErrMalformedHTTPSOrigin = errors.New("the address is not an https origin")

type HTTPSOrigin struct {
	value string
}

type originFault struct {
	reason string
	holds  bool
}

func ParseHTTPSOrigin(stated string) (HTTPSOrigin, error) {
	parsed, err := url.Parse(stated)
	if err != nil {
		return HTTPSOrigin{}, fmt.Errorf("%w: %w", ErrMalformedHTTPSOrigin, err)
	}
	if fault, isFaulty := originFaultOf(parsed); isFaulty {
		return HTTPSOrigin{}, fmt.Errorf("%w: %q %s", ErrMalformedHTTPSOrigin, stated, fault)
	}
	return HTTPSOrigin{value: stated}, nil
}

func originFaultOf(address *url.URL) (string, bool) {
	faults := []originFault{
		{"is not https", address.Scheme != httpsScheme},
		{"names no host", address.Host == ""},
		{"carries a path", address.Path != "" || address.RawPath != "" || address.Opaque != ""},
		{"carries a query", address.RawQuery != "" || address.ForceQuery},
		{"carries a fragment", address.Fragment != ""},
		{"carries credentials", address.User != nil},
	}
	held := slices.IndexFunc(faults, func(fault originFault) bool { return fault.holds })
	if held < 0 {
		return "", false
	}
	return faults[held].reason, true
}
