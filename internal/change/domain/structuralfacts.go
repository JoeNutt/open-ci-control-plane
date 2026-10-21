// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"encoding/json"
	"errors"
	"fmt"
)

var ErrTooManyImportsInChange = errors.New("change exceeds the per-change import bound")

var ErrTooManyEdges = errors.New("change exceeds the module graph edge bound")

var ErrStructuralFactsTooLarge = errors.New("structural facts exceed the serialised byte bound")

type StructuralFacts struct {
	imports      Declared[Import]
	edgesAdded   Declared[ModuleEdge]
	edgesRemoved Declared[ModuleEdge]
}

func ParseStructuralFacts(imported []ImportedFile, limits Limits) (StructuralFacts, error) {
	published := headImportCountOf(imported)
	if published > limits.MaxImportsPerChange {
		return StructuralFacts{}, fmt.Errorf("%w: %d imports exceeds %d",
			ErrTooManyImportsInChange, published, limits.MaxImportsPerChange)
	}
	facts := StructuralFacts{
		imports:      importsOf(imported),
		edgesAdded:   edgesIn(imported, ImportedFile.edgesAdded),
		edgesRemoved: edgesIn(imported, ImportedFile.edgesRemoved),
	}
	if edges := facts.edgeCount(); edges > limits.MaxGraphEdges {
		return StructuralFacts{}, fmt.Errorf("%w: %d edges exceeds %d",
			ErrTooManyEdges, edges, limits.MaxGraphEdges)
	}
	serialised, err := json.Marshal(struct {
		Imports          Declared[Import] `json:"imports"`
		ModuleGraphDelta ModuleGraphDelta `json:"module_graph_delta"`
	}{facts.imports, graphDeltaFor(importResolutionIn(facts.imports), facts)})
	if err != nil {
		return StructuralFacts{}, err
	}
	if len(serialised) > limits.MaxStructuralBytes {
		return StructuralFacts{}, fmt.Errorf("%w: %d bytes exceeds %d",
			ErrStructuralFactsTooLarge, len(serialised), limits.MaxStructuralBytes)
	}
	return facts, nil
}

func (facts StructuralFacts) edgeCount() int {
	return len(facts.edgesAdded) + len(facts.edgesRemoved)
}

func headImportCountOf(imported []ImportedFile) int {
	count := 0
	for _, file := range imported {
		count += len(file.Head.imports)
	}
	return count
}

func importsOf(imported []ImportedFile) Declared[Import] {
	imports := make(Declared[Import], 0, len(imported))
	for _, file := range imported {
		if file.existsAtHead() {
			imports = append(imports, file.headImport())
		}
	}
	return imports
}

func edgesIn(
	imported []ImportedFile, edgesOf func(ImportedFile) []ModuleEdge,
) Declared[ModuleEdge] {
	var edges Declared[ModuleEdge]
	for _, file := range imported {
		edges = append(edges, edgesOf(file)...)
	}
	return edges
}
