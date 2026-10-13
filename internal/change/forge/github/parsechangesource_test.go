// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
)

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
