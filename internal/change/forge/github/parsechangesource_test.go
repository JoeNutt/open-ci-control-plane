// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"errors"
	"net/http"
	"testing"

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
			_, err := github.ParseChangeSource(github.Settings{
				BaseURL:          address,
				Token:            forgeCredential,
				Client:           http.DefaultClient,
				MaxResponseBytes: 1 << 20,
			})

			if !errors.Is(err, github.ErrMalformedBaseURL) {
				t.Fatalf("error is %v, want %v", err, github.ErrMalformedBaseURL)
			}
		})
	}
}

func TestParseChangeSourceRefusesASourceWithNoClient(t *testing.T) {
	_, err := github.ParseChangeSource(github.Settings{
		BaseURL:          "https://api.example",
		Token:            forgeCredential,
		MaxResponseBytes: 1 << 20,
	})

	if !errors.Is(err, github.ErrNoClient) {
		t.Fatalf("error is %v, want %v", err, github.ErrNoClient)
	}
}

func TestParseChangeSourceRefusesASourceWithNoCredential(t *testing.T) {
	_, err := github.ParseChangeSource(github.Settings{
		BaseURL:          "https://api.example",
		Client:           http.DefaultClient,
		MaxResponseBytes: 1 << 20,
	})

	if !errors.Is(err, github.ErrNoCredential) {
		t.Fatalf("error is %v, want %v", err, github.ErrNoCredential)
	}
}
