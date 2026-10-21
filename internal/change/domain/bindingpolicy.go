// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"slices"
)

type BoundPolicyStatement struct {
	Use         string
	ImportRoots []ImportRootStatement
	Paths       map[string][]string
}

type BoundPolicy struct {
	use         PolicyName
	importRoots []ImportRoot
	groups      []GroupPaths
}

func parseBoundPolicy(stated BoundPolicyStatement) (BoundPolicy, error) {
	use, err := ParsePolicyName(stated.Use)
	if err != nil {
		return BoundPolicy{}, fmt.Errorf("bound policy: %w", err)
	}
	roots, err := importRootsOf(stated.ImportRoots)
	if err != nil {
		return BoundPolicy{}, fmt.Errorf("bound policy %s: %w", use, err)
	}
	groups, err := groupPathsOf(stated.Paths)
	if err != nil {
		return BoundPolicy{}, fmt.Errorf("bound policy %s: %w", use, err)
	}
	return BoundPolicy{use: use, importRoots: roots, groups: groups}, nil
}

func (policy BoundPolicy) Groups() []GroupPaths { return slices.Clone(policy.groups) }

func (policy BoundPolicy) Use() PolicyName { return policy.use }

func (policy BoundPolicy) ImportRoots() []ImportRoot { return slices.Clone(policy.importRoots) }
