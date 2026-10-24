// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"reflect"
	"testing"
)

func TestChangeEvidenceCarriesEveryChangedFile(t *testing.T) {
	changed := changedFileFrom(t, StatedFile{
		File:  File{Path: "internal/auth/owner.go", Action: FileActionModified},
		Patch: "@@ -1,1 +1,1 @@\n-old\n+new", IsPatchStated: true,
	})

	evidence, err := ParseChangeEvidence([]ChangedFile{changed}, nil, bindingRootedAt(t, "example.com/app"))

	if err != nil {
		t.Fatalf("parse change evidence: %v", err)
	}
	if !reflect.DeepEqual(evidence.files, []ChangedFile{changed}) {
		t.Fatalf("files are %+v, want %+v", evidence.files, changed)
	}
}

func bindingRootedAt(t *testing.T, module string) Binding {
	t.Helper()

	binding, err := ParseBinding(BindingStatement{
		Repository: "acme/app",
		Workflow:   "tests.yml",
		Policies: []BoundPolicyStatement{{
			Use:         "clean-architecture",
			ImportRoots: []ImportRootStatement{{Prefix: module, Path: ""}},
		}},
	})
	if err != nil {
		t.Fatalf("parse binding: %v", err)
	}
	return binding
}
