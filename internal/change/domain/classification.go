// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"slices"
)

const MaxPatternsPerKind = 16

var ErrTooManyPatternsForKind = errors.New("classification rules give a kind more patterns than it may hold")

var ErrUnknownPathKind = errors.New("classification rules name a kind the schema does not declare")

type ClassificationRules struct {
	patterns map[PathKind][]PathPattern
}

func ParseClassificationRules(stated map[PathKind][]string) (ClassificationRules, error) {
	patterns := make(map[PathKind][]PathPattern, len(stated))
	for kind, written := range stated {
		if !slices.Contains(declaredPathKinds, kind) {
			return ClassificationRules{}, fmt.Errorf("%w: %q", ErrUnknownPathKind, kind)
		}
		if len(written) > MaxPatternsPerKind {
			return ClassificationRules{}, fmt.Errorf("%w: %q holds %d, which exceeds %d",
				ErrTooManyPatternsForKind, kind, len(written), MaxPatternsPerKind)
		}
		for _, one := range written {
			parsed, err := ParsePathPattern(one)
			if err != nil {
				return ClassificationRules{}, fmt.Errorf("kind %q: %w", kind, err)
			}
			patterns[kind] = append(patterns[kind], parsed)
		}
	}
	return ClassificationRules{patterns: patterns}, nil
}

func (rules ClassificationRules) KindsOf(paths Paths) Kinds {
	return Kinds{
		Test:         rules.matching(PathKindTest, paths),
		Authz:        rules.matching(PathKindAuthz, paths),
		Route:        rules.matching(PathKindRoute, paths),
		IaC:          rules.matching(PathKindIaC, paths),
		CIConfig:     rules.matching(PathKindCIConfig, paths),
		PolicyConfig: rules.matching(PathKindPolicyConfig, paths),
		Lockfile:     rules.matching(PathKindLockfile, paths),
		Migration:    rules.matching(PathKindMigration, paths),
		Generated:    rules.matching(PathKindGenerated, paths),
	}
}

func (rules ClassificationRules) matching(kind PathKind, paths Paths) Paths {
	matched := make(Paths, 0, len(paths))
	for _, candidate := range paths {
		if slices.ContainsFunc(rules.patterns[kind], matcher(candidate)) {
			matched = append(matched, candidate)
		}
	}
	return matched
}

func matcher(candidate string) func(PathPattern) bool {
	return func(pattern PathPattern) bool { return pattern.Matches(candidate) }
}
