// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type Settings struct {
	BaseURL          string
	Token            Credential
	Client           *http.Client
	MaxResponseBytes int
}

var ErrNoCredential = errors.New("a change source is given no forge credential")

var ErrMalformedBaseURL = errors.New("the forge address is not an https origin")

func ParseChangeSource(settings Settings) (ChangeSource, error) {
	if settings.Token == "" {
		return ChangeSource{}, ErrNoCredential
	}
	if err := checkBaseURL(settings.BaseURL); err != nil {
		return ChangeSource{}, err
	}
	return ChangeSource{settings: settings}, nil
}

func checkBaseURL(address string) error {
	parsed, err := url.Parse(address)
	if err != nil {
		return fmt.Errorf("%w: %q: %w", ErrMalformedBaseURL, address, err)
	}
	faults := []struct {
		reason string
		holds  bool
	}{
		{"is not https", parsed.Scheme != "https"},
		{"names no host", parsed.Host == ""},
		{"ends in a slash", strings.HasSuffix(parsed.Path, "/")},
		{"carries a query", parsed.RawQuery != ""},
		{"carries a fragment", parsed.Fragment != ""},
	}
	for _, fault := range faults {
		if fault.holds {
			return fmt.Errorf("%w: %q %s", ErrMalformedBaseURL, address, fault.reason)
		}
	}
	return nil
}
