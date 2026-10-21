// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"maps"
	"slices"
)

var ErrPathInTwoGroups = errors.New("binding places one path in two groups")

type GroupPaths struct {
	group    GroupName
	patterns []BoundGlob
}

func (paths GroupPaths) Group() GroupName { return paths.group }

func (paths GroupPaths) Patterns() []BoundGlob { return slices.Clone(paths.patterns) }

func groupPathsOf(stated map[string][]string) ([]GroupPaths, error) {
	groups := make([]GroupPaths, 0, len(stated))
	for _, group := range slices.Sorted(maps.Keys(stated)) {
		parsed, err := parseGroupPaths(group, stated[group])
		if err != nil {
			return nil, err
		}
		if earlier := slices.IndexFunc(groups, parsed.sharesAPatternWith); earlier >= 0 {
			return nil, fmt.Errorf("%w: %s and %s", ErrPathInTwoGroups, groups[earlier].group, parsed.group)
		}
		groups = append(groups, parsed)
	}
	return groups, nil
}

func parseGroupPaths(stated string, written []string) (GroupPaths, error) {
	group, err := ParseGroupName(stated)
	if err != nil {
		return GroupPaths{}, fmt.Errorf("paths: %w", err)
	}
	patterns, err := boundGlobsOf(written)
	if err != nil {
		return GroupPaths{}, fmt.Errorf("paths of %s: %w", group, err)
	}
	return GroupPaths{group: group, patterns: patterns}, nil
}

func (paths GroupPaths) sharesAPatternWith(other GroupPaths) bool {
	return slices.ContainsFunc(paths.patterns, other.holds)
}

func (paths GroupPaths) holds(glob BoundGlob) bool {
	return slices.ContainsFunc(paths.patterns, glob.translatesLike)
}

func boundGlobsOf(written []string) ([]BoundGlob, error) {
	globs := make([]BoundGlob, 0, len(written))
	for _, glob := range written {
		parsed, err := parseBoundGlob(glob)
		if err != nil {
			return nil, err
		}
		globs = append(globs, parsed)
	}
	return globs, nil
}
