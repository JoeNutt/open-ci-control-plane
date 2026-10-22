// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "slices"

type KnownPaths struct {
	paths []RepositoryPath
}

func NewKnownPaths(paths []RepositoryPath) KnownPaths {
	return KnownPaths{paths: slices.Clone(paths)}
}

func ParseKnownPaths(stated []string) (KnownPaths, error) {
	paths := make([]RepositoryPath, 0, len(stated))
	for _, statedPath := range stated {
		path, err := ParseRepositoryPath(statedPath)
		if err != nil {
			return KnownPaths{}, err
		}
		paths = append(paths, path)
	}
	return KnownPaths{paths: paths}, nil
}

func (known KnownPaths) Contains(path RepositoryPath) bool {
	return slices.Contains(known.paths, path)
}
