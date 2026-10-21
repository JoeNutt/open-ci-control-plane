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

func TestACredentialMarshalsToJSONAsTheRedactionMarker(t *testing.T) {
	encoded, err := json.Marshal(forgeCredential)

	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(encoded) != strconv.Quote(github.RedactionMarker) {
		t.Errorf("a credential marshals to %s, want %q", encoded, github.RedactionMarker)
	}
}

func TestACredentialPrintsOnlyTheRedactionMarkerUnderEveryVerb(t *testing.T) {
	for _, verb := range everyFormatVerb {
		if slices.Contains(verbsFmtAnswersBeforeAnyMethod, verb) {
			continue
		}
		if printed := fmt.Sprintf(verb, forgeCredential); printed != github.RedactionMarker {
			t.Errorf("a credential under %s printed %q, want %q", verb, printed, github.RedactionMarker)
		}
	}
}
