// SPDX-License-Identifier: GPL-3.0-or-later

package policy_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/policy"
)

func TestParseRevisionParsesTheLivePolicyRepository(t *testing.T) {
	model, err := policy.ParseRevision(liveFiles(t))
	if err != nil {
		t.Fatalf("parse revision: %v", err)
	}
	if len(model.Catalog()) != 1 || len(model.Bindings()) != 1 {
		t.Errorf("model holds %d catalog policies and %d bindings, want one of each",
			len(model.Catalog()), len(model.Bindings()))
	}
}

func TestParseRevisionRefusesAFileOutsideTheCatalogAndBindings(t *testing.T) {
	files := liveFiles(t)
	files[1].Path = "templates/JoeNutt/open-ci-control-plane.yaml"

	_, err := policy.ParseRevision(files)
	if !errors.Is(err, policy.ErrUnplacedPolicyFile) {
		t.Fatalf("a binding under templates/ gave %v, want ErrUnplacedPolicyFile", err)
	}
}

func liveFiles(t *testing.T) []domain.PolicyFile {
	t.Helper()

	return []domain.PolicyFile{
		{Path: liveCatalogFile, Content: fixture(t, liveCatalogPath)},
		{Path: liveBindingFile, Content: fixture(t, liveBindingPath)},
	}
}
