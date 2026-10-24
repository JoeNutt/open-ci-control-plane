// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type indexSize struct {
	tests int
	edges int
}

func (size indexSize) replacing(held, replacement indexSize) indexSize {
	return indexSize{
		tests: size.tests - held.tests + replacement.tests,
		edges: size.edges - held.edges + replacement.edges,
	}
}

func (size indexSize) breachIn(limits CoverageLimits) error {
	if breach := limits.indexedTestsBreach(size.tests); breach != nil {
		return breach
	}
	return limits.indexEdgesBreach(size.edges)
}
