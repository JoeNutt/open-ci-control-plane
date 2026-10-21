// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
)

var verbsFmtAnswersBeforeAnyMethod = []string{"%T", "%p"}

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
