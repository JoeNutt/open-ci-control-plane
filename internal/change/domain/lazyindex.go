// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "sync"

var emptyCoverageIndex = indexOf(nil)

type lazyIndex struct {
	suites map[CoverageSuite]SuiteCoverage
	once   sync.Once
	built  coverageIndex
}

func (lazy *lazyIndex) index() coverageIndex {
	if lazy == nil {
		return emptyCoverageIndex
	}
	lazy.once.Do(func() { lazy.built = indexOf(lazy.suites) })
	return lazy.built
}
