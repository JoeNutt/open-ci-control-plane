// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseAuthorAssociationRefusesAnUnmappedValue(t *testing.T) {
	association, err := domain.ParseAuthorAssociation("MAINTAINER")

	if !errors.Is(err, domain.ErrUnknownAuthorAssociation) {
		t.Fatalf("want ErrUnknownAuthorAssociation, got %v", err)
	}
	if association != "" {
		t.Fatalf("want no association alongside the refusal, got %q", association)
	}
}

func TestParseAuthorAssociationAcceptsEveryDeclaredAssociation(t *testing.T) {
	declared := map[string]domain.AuthorAssociation{
		"OWNER":                  domain.AuthorAssociationOwner,
		"MEMBER":                 domain.AuthorAssociationMember,
		"COLLABORATOR":           domain.AuthorAssociationCollaborator,
		"CONTRIBUTOR":            domain.AuthorAssociationContributor,
		"FIRST_TIME_CONTRIBUTOR": domain.AuthorAssociationFirstTimeContributor,
		"FIRST_TIMER":            domain.AuthorAssociationFirstTimer,
		"MANNEQUIN":              domain.AuthorAssociationMannequin,
		"NONE":                   domain.AuthorAssociationNone,
	}

	for name, want := range declared {
		t.Run(name, func(t *testing.T) {
			association, err := domain.ParseAuthorAssociation(name)
			if err != nil {
				t.Fatalf("parse %q: %v", name, err)
			}
			if association != want {
				t.Fatalf("parsed %q as %q, want %q", name, association, want)
			}
		})
	}
}
