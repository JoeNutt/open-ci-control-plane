// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
)

func TestParseCheckRunsRefusesWhatAChangeSourceRefuses(t *testing.T) {
	refusals := map[string]struct {
		spoil func(*github.Settings)
		want  error
	}{
		"no credential": {func(s *github.Settings) { s.Token = "" }, github.ErrNoCredential},
		"no client":     {func(s *github.Settings) { s.Client = nil }, github.ErrNoClient},
		"plain http": {
			func(s *github.Settings) { s.BaseURL = "http://api.example" },
			github.ErrMalformedBaseURL,
		},
	}

	for name, refusal := range refusals {
		t.Run(name, func(t *testing.T) {
			settings := wellFormedSettings()
			refusal.spoil(&settings)

			_, err := github.ParseCheckRuns(settings)

			if !errors.Is(err, refusal.want) {
				t.Fatalf("error is %v, want %v", err, refusal.want)
			}
		})
	}
}
