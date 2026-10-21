// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"maps"
	"slices"
)

type GroupPaths struct {
	group    GroupName
	patterns []PathPattern
}

func (paths GroupPaths) Group() GroupName { return paths.group }

func (paths GroupPaths) Patterns() []PathPattern { return slices.Clone(paths.patterns) }

func groupPathsOf(stated map[string][]string) ([]GroupPaths, error) {
	groups := make([]GroupPaths, 0, len(stated))
	for _, group := range slices.Sorted(maps.Keys(stated)) {
		parsed, err := parseGroupPaths(group, stated[group])
		if err != nil {
			return nil, err
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
	patterns, err := pathPatternsOf(written)
	if err != nil {
		return GroupPaths{}, fmt.Errorf("paths of %s: %w", group, err)
	}
	return GroupPaths{group: group, patterns: patterns}, nil
}

func pathPatternsOf(written []string) ([]PathPattern, error) {
	patterns := make([]PathPattern, 0, len(written))
	for _, pattern := range written {
		parsed, err := ParsePathPattern(pattern)
		if err != nil {
			return nil, err
		}
		patterns = append(patterns, parsed)
	}
	return patterns, nil
}
