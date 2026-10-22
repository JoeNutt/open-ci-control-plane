// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"net/url"
	"slices"
)

const httpsScheme = "https"

var ErrMalformedHTTPSOrigin = errors.New("the URL is not an https origin")

type HTTPSOrigin struct {
	origin *url.URL
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
		{reason: "its scheme is not https", holds: parsed.Scheme != httpsScheme},
		{reason: "it names no host", holds: parsed.Host == ""},
		{reason: "it carries user information", holds: parsed.User != nil},
		{reason: "it carries a path", holds: parsed.Path != "" || parsed.RawPath != ""},
		{reason: "it carries a query", holds: parsed.RawQuery != "" || parsed.ForceQuery},
		{reason: "it carries a fragment", holds: parsed.Fragment != ""},
	}
	held := slices.IndexFunc(faults, func(fault originFault) bool { return fault.holds })
	if held >= 0 {
		return HTTPSOrigin{}, fmt.Errorf("%w: %s", ErrMalformedHTTPSOrigin, faults[held].reason)
	}
	return HTTPSOrigin{origin: parsed}, nil
}

func (origin HTTPSOrigin) String() string { return origin.origin.String() }
