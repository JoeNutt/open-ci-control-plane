// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "slices"

type TestedFile struct {
	path     RepositoryPath
	hitLines []uint32
}

func NewTestedFile(path RepositoryPath, hitLines []uint32) TestedFile {
	return TestedFile{path: path, hitLines: slices.Compact(slices.Sorted(slices.Values(hitLines)))}
}

func (file TestedFile) joinedWith(other TestedFile) TestedFile {
	return NewTestedFile(file.path, slices.Concat(file.hitLines, other.hitLines))
}
