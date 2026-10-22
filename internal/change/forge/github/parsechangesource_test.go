// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
)

func TestParseChangeSourceRefusesAnAddressThatIsNotAnHTTPSOrigin(t *testing.T) {
	malformed := []string{
		"",
		"api.example",
		"http://api.example",
		"https://",
		"https://api.example/",
		"https://api.example/api/v3/",
		"https://api.example?per_page=1",
		"https://api.example#fragment",
	}

	for _, address := range malformed {
		t.Run(address, func(t *testing.T) {
			settings := wellFormedSettings()
			settings.BaseURL = address

			_, err := github.ParseChangeSource(settings)

			if !errors.Is(err, github.ErrMalformedBaseURL) {
				t.Fatalf("error is %v, want %v", err, github.ErrMalformedBaseURL)
			}
		})
	}
}

func TestParsingRefusesAResponseBoundThatIsNotPositive(t *testing.T) {
	for _, bound := range []int{0, -1} {
		settings := wellFormedSettings()
		settings.MaxResponseBytes = bound

		_, sourceErr := github.ParseChangeSource(settings)
		_, runsErr := github.ParseCheckRuns(settings)

		for parser, err := range map[string]error{"change source": sourceErr, "check runs": runsErr} {
			if !errors.Is(err, github.ErrMalformedResponseBound) {
				t.Errorf("%s with bound %d: error is %v, want %v",
					parser, bound, err, github.ErrMalformedResponseBound)
			}
		}
	}
}

func TestParseChangeSourceRefusesASourceWithNoClient(t *testing.T) {
	settings := wellFormedSettings()
	settings.Client = nil

	_, err := github.ParseChangeSource(settings)

	if !errors.Is(err, github.ErrNoClient) {
		t.Fatalf("error is %v, want %v", err, github.ErrNoClient)
	}
}

func TestParseChangeSourceRefusesASourceWithNoCredential(t *testing.T) {
	settings := wellFormedSettings()
	settings.Credentials = github.Credential{}

	_, err := github.ParseChangeSource(settings)

	if !errors.Is(err, github.ErrNoCredential) {
		t.Fatalf("error is %v, want %v", err, github.ErrNoCredential)
	}
}

func TestParseChangeSourceRefusesAClientWithNoTimeout(t *testing.T) {
	for _, timeout := range []time.Duration{0, -time.Second} {
		settings := wellFormedSettings()
		settings.Client = &http.Client{Timeout: timeout}

		_, err := github.ParseChangeSource(settings)

		if !errors.Is(err, github.ErrUnboundedClient) {
			t.Errorf("timeout %v: error is %v, want %v", timeout, err, github.ErrUnboundedClient)
		}
	}
}
