// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "slices"

type KnownPaths struct {
	paths []RepositoryPath
}

func NewKnownPaths(paths []RepositoryPath) KnownPaths {
	return KnownPaths{paths: paths}
}

func (known KnownPaths) Contains(path RepositoryPath) bool {
	return slices.Contains(known.paths, path)
}
