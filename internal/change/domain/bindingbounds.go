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
	count := 0
	for _, policy := range stated.Policies {
		count += len(policy.Paths)
	}
	return count
}

func pathPatternCountOf(stated BindingStatement) int {
	count := 0
	for _, policy := range stated.Policies {
		for _, patterns := range policy.Paths {
			count += len(patterns)
		}
	}
	return count
}

func importRootCountOf(stated BindingStatement) int {
	count := 0
	for _, policy := range stated.Policies {
		count += len(policy.ImportRoots)
	}
	return count
}

func alwaysRunCountOf(stated BindingStatement) int { return len(stated.AlwaysRun) }
