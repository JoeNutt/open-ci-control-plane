// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
)

const (
	MaxBindingGroups          = 64
	MaxBindingPathPatterns    = 512
	MaxBindingImportRoots     = 64
	MaxBindingAlwaysRunSuites = 512
)

var ErrTooManyBindingGroups = errors.New("binding exceeds the module group bound")

var ErrTooManyBindingPathPatterns = errors.New("binding exceeds the path pattern bound")

var ErrTooManyBindingImportRoots = errors.New("binding exceeds the import root bound")

var ErrTooManyAlwaysRunSuites = errors.New("binding exceeds the always-run suite bound")

var boundsOfABinding = []struct {
	measurementOf func(BindingStatement) int
	allowance     int
	breach        error
}{
	{groupCountOf, MaxBindingGroups, ErrTooManyBindingGroups},
	{pathPatternCountOf, MaxBindingPathPatterns, ErrTooManyBindingPathPatterns},
	{importRootCountOf, MaxBindingImportRoots, ErrTooManyBindingImportRoots},
	{alwaysRunCountOf, MaxBindingAlwaysRunSuites, ErrTooManyAlwaysRunSuites},
}

func bindingBoundBreachIn(stated BindingStatement) error {
	for _, bound := range boundsOfABinding {
		if measured := bound.measurementOf(stated); measured > bound.allowance {
			return fmt.Errorf("%w: %d exceeds %d", bound.breach, measured, bound.allowance)
		}
	}
	return nil
}

func groupCountOf(stated BindingStatement) int {
	return countAcrossPolicies(stated, func(policy BoundPolicyStatement) int { return len(policy.Paths) })
}

func pathPatternCountOf(stated BindingStatement) int {
	return countAcrossPolicies(stated, patternCountOf)
}

func importRootCountOf(stated BindingStatement) int {
	return countAcrossPolicies(stated, func(policy BoundPolicyStatement) int { return len(policy.ImportRoots) })
}

func alwaysRunCountOf(stated BindingStatement) int { return len(stated.AlwaysRun) }

func countAcrossPolicies(stated BindingStatement, countIn func(BoundPolicyStatement) int) int {
	count := 0
	for _, policy := range stated.Policies {
		count += countIn(policy)
	}
	return count
}

func patternCountOf(policy BoundPolicyStatement) int {
	count := 0
	for _, patterns := range policy.Paths {
		count += len(patterns)
	}
	return count
}
