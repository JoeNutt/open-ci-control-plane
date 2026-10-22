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
	faults := []originFault{
		{"is not https", parsed.Scheme != httpsScheme},
		{"names no host", parsed.Host == ""},
		{"carries a path", parsed.Path != "" || parsed.RawPath != "" || parsed.Opaque != ""},
		{"carries a query", parsed.RawQuery != "" || parsed.ForceQuery},
		{"carries a fragment", parsed.Fragment != ""},
		{"carries credentials", parsed.User != nil},
	}
	held := slices.IndexFunc(faults, func(fault originFault) bool { return fault.holds })
	if held >= 0 {
		return HTTPSOrigin{}, fmt.Errorf("%w: %q %s", ErrMalformedHTTPSOrigin, stated, faults[held].reason)
	}
	return HTTPSOrigin{value: stated}, nil
}
