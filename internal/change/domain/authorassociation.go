// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"slices"
)

type AuthorAssociation string

const (
	AuthorAssociationOwner                AuthorAssociation = "OWNER"
	AuthorAssociationMember               AuthorAssociation = "MEMBER"
	AuthorAssociationCollaborator         AuthorAssociation = "COLLABORATOR"
	AuthorAssociationContributor          AuthorAssociation = "CONTRIBUTOR"
	AuthorAssociationFirstTimeContributor AuthorAssociation = "FIRST_TIME_CONTRIBUTOR"
	AuthorAssociationFirstTimer           AuthorAssociation = "FIRST_TIMER"
	AuthorAssociationMannequin            AuthorAssociation = "MANNEQUIN"
	AuthorAssociationNone                 AuthorAssociation = "NONE"
)

var declaredAuthorAssociations = []AuthorAssociation{
	AuthorAssociationOwner,
	AuthorAssociationMember,
	AuthorAssociationCollaborator,
	AuthorAssociationContributor,
	AuthorAssociationFirstTimeContributor,
	AuthorAssociationFirstTimer,
	AuthorAssociationMannequin,
	AuthorAssociationNone,
}

var ErrUnknownAuthorAssociation = errors.New("unknown author association")

func ParseAuthorAssociation(name string) (AuthorAssociation, error) {
	association := AuthorAssociation(name)
	if slices.Contains(declaredAuthorAssociations, association) {
		return association, nil
	}
	return "", fmt.Errorf("%w: %q", ErrUnknownAuthorAssociation, name)
}
