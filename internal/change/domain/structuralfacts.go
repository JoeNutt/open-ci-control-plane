// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"slices"
)

var ErrTooManyImportsInChange = errors.New("change exceeds the per-change import bound")

var ErrTooManyEdges = errors.New("change exceeds the module graph edge bound")

type StructuralFacts struct{ imported []ImportedFile }

func ParseStructuralFacts(imported []ImportedFile, limits Limits) (StructuralFacts, error) {
	published := headImportCountOf(imported)
	if published > limits.MaxImportsPerChange {
		return StructuralFacts{}, fmt.Errorf("%w: %d imports exceeds %d",
			ErrTooManyImportsInChange, published, limits.MaxImportsPerChange)
	}
	edges := len(edgesIn(imported, ImportedFile.edgesAdded)) + len(edgesIn(imported, ImportedFile.edgesRemoved))
	if edges > limits.MaxGraphEdges {
		return StructuralFacts{}, fmt.Errorf("%w: %d edges exceeds %d", ErrTooManyEdges, edges, limits.MaxGraphEdges)
	}
	return StructuralFacts{imported: slices.Clone(imported)}, nil
}

func headImportCountOf(imported []ImportedFile) int {
	count := 0
	for _, file := range imported {
		count += len(file.Head.imports)
	}
	return count
}
