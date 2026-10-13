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
	malformed := parsed.Scheme != "https" ||
		parsed.Host == "" ||
		strings.HasSuffix(parsed.Path, "/") ||
		parsed.RawQuery != "" ||
		parsed.Fragment != ""
	if malformed {
		return fmt.Errorf("%w: %q", ErrMalformedBaseURL, address)
	}
	return nil
}
