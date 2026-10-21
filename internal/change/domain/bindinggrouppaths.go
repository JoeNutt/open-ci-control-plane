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
	for _, written := range slices.Sorted(maps.Keys(stated)) {
		group, err := ParseGroupName(written)
		if err != nil {
			return nil, fmt.Errorf("paths: %w", err)
		}
		patterns := make([]PathPattern, 0, len(stated[written]))
		for _, pattern := range stated[written] {
			parsed, err := ParsePathPattern(pattern)
			if err != nil {
				return nil, fmt.Errorf("paths of %s: %w", group, err)
			}
			patterns = append(patterns, parsed)
		}
		groups = append(groups, GroupPaths{group: group, patterns: patterns})
	}
	return groups, nil
}
