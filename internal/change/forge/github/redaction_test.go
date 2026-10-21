// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
)

var verbsFmtAnswersBeforeAnyMethod = []string{"%T", "%p"}

func secretHolders(t *testing.T) map[string]any {
	t.Helper()

	return map[string]any{
		"credential": forgeCredential,
		"App key":    forgeAppKey(t),
	}
}

func TestASecretMarshalsToJSONAsTheRedactionMarker(t *testing.T) {
	for name, secret := range secretHolders(t) {
		encoded, err := json.Marshal(secret)

		if err != nil {
			t.Fatalf("marshal the %s: %v", name, err)
		}
		if string(encoded) != strconv.Quote(github.RedactionMarker) {
			t.Errorf("the %s marshals to %.80s, want %q", name, encoded, github.RedactionMarker)
		}
	}
}

func TestASecretPrintsOnlyTheRedactionMarkerUnderEveryVerb(t *testing.T) {
	for name, secret := range secretHolders(t) {
		for _, verb := range everyFormatVerb {
			if slices.Contains(verbsFmtAnswersBeforeAnyMethod, verb) {
				continue
			}
			if printed := fmt.Sprintf(verb, secret); printed != github.RedactionMarker {
				t.Errorf("the %s under %s printed %.80q, want %q", name, verb, printed, github.RedactionMarker)
			}
		}
	}
}
