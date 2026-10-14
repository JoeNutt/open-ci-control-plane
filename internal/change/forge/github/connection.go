// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
)

type Settings struct {
	BaseURL          string
	Token            Credential
	AppID            int64
	Client           *http.Client
	MaxResponseBytes int
}

var ErrNoCredential = errors.New("a change source is given no forge credential")

var ErrMalformedBaseURL = errors.New("the forge address is not an https origin")

var ErrMalformedResponseBound = errors.New("the forge settings carry no positive response byte bound")

var ErrNoClient = errors.New("a change source is given no http client")

type connection struct {
	settings Settings
	token    *Credential
}

func parseConnection(settings Settings) (connection, error) {
	if settings.Token == "" {
		return connection{}, ErrNoCredential
	}
	if settings.Client == nil {
		return connection{}, ErrNoClient
	}
	if err := checkBaseURL(settings.BaseURL); err != nil {
		return connection{}, err
	}
	if settings.MaxResponseBytes <= 0 {
		return connection{}, fmt.Errorf("%w: %d", ErrMalformedResponseBound, settings.MaxResponseBytes)
	}
	token := settings.Token
	settings.Token = ""
	return connection{settings: settings, token: &token}, nil
}

func checkBaseURL(address string) error {
	parsed, err := url.Parse(address)
	if err != nil {
		return fmt.Errorf("%w: %q: %w", ErrMalformedBaseURL, address, err)
	}
	faults := []baseURLFault{
		{"is not https", parsed.Scheme != "https"},
		{"names no host", parsed.Host == ""},
		{"ends in a slash", strings.HasSuffix(parsed.Path, "/")},
		{"carries a query", parsed.RawQuery != ""},
		{"carries a fragment", parsed.Fragment != ""},
	}
	held := slices.IndexFunc(faults, func(fault baseURLFault) bool { return fault.holds })
	if held < 0 {
		return nil
	}
	return fmt.Errorf("%w: %q %s", ErrMalformedBaseURL, address, faults[held].reason)
}

type baseURLFault struct {
	reason string
	holds  bool
}
