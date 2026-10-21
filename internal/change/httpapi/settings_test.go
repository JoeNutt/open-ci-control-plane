// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/httpapi"
)

func TestParseRoutesRefusesSettingsMissingAPart(t *testing.T) {
	incomplete := map[string]func(*httpapi.Settings){
		"no log":                  func(settings *httpapi.Settings) { settings.Log = nil },
		"no gate":                 func(settings *httpapi.Settings) { settings.Gate = nil },
		"no decision ID source":   func(settings *httpapi.Settings) { settings.DecisionIDs = nil },
		"no allowed installation": func(settings *httpapi.Settings) { settings.Installations = nil },
		"no clock":                func(settings *httpapi.Settings) { settings.Clock = nil },
	}
	for form, without := range incomplete {
		t.Run(form, func(t *testing.T) {
			settings := listenerSettings(discardedLog(), &recordingGate{})
			without(&settings)

			_, err := httpapi.ParseRoutes(settings)

			if !errors.Is(err, httpapi.ErrIncompleteRoutes) {
				t.Errorf("error is %v, want %v", err, httpapi.ErrIncompleteRoutes)
			}
		})
	}
}
