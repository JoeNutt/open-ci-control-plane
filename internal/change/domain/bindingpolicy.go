// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"slices"
)

var ErrDuplicateBoundPolicy = errors.New("binding uses one catalog policy twice")

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

func (policy BoundPolicy) Use() PolicyName { return policy.use }

func (policy BoundPolicy) ImportRoots() []ImportRoot { return slices.Clone(policy.importRoots) }

func (policy BoundPolicy) Groups() []GroupPaths { return slices.Clone(policy.groups) }

func boundPoliciesOf(stated []BoundPolicyStatement) ([]BoundPolicy, error) {
	policies := make([]BoundPolicy, 0, len(stated))
	for _, written := range stated {
		policy, err := parseBoundPolicy(written)
		if err != nil {
			return nil, err
		}
		if slices.ContainsFunc(policies, policy.usesSamePolicyAs) {
			return nil, fmt.Errorf("%w: %s", ErrDuplicateBoundPolicy, policy.use)
		}
		policies = append(policies, policy)
	}
	return policies, nil
}

func parseBoundPolicy(stated BoundPolicyStatement) (BoundPolicy, error) {
	use, err := ParsePolicyName(stated.Use)
	if err != nil {
		return BoundPolicy{}, fmt.Errorf("bound policy: %w", err)
	}
	policy, err := parseBoundPolicyNamed(use, stated)
	if err != nil {
		return BoundPolicy{}, fmt.Errorf("bound policy %s: %w", use, err)
	}
	return policy, nil
}

func parseBoundPolicyNamed(use PolicyName, stated BoundPolicyStatement) (BoundPolicy, error) {
	roots, err := importRootsOf(stated.ImportRoots)
	if err != nil {
		return BoundPolicy{}, err
	}
	groups, err := groupPathsOf(stated.Paths)
	if err != nil {
		return BoundPolicy{}, err
	}
	return BoundPolicy{use: use, importRoots: roots, groups: groups}, nil
}

func (policy BoundPolicy) usesSamePolicyAs(other BoundPolicy) bool { return policy.use == other.use }
