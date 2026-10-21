// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

var ErrTooManyImportsInChange = errors.New("change exceeds the per-change import bound")

var ErrTooManyEdges = errors.New("change exceeds the module graph edge bound")

var ErrStructuralFactsTooLarge = errors.New("structural facts exceed the serialised byte bound")

type StructuralFacts struct {
	imports      Declared[Import]
	edgesAdded   Declared[ModuleEdge]
	edgesRemoved Declared[ModuleEdge]
}

var boundsOfStructuralFacts = []func(StructuralFacts, Limits) error{
	edgeCountBreachIn,
	serialisedBytesBreachIn,
}

func ParseStructuralFacts(imported []ImportedFile, limits Limits) (StructuralFacts, error) {
	if breach := changeImportCountBreachIn(imported, limits); breach != nil {
		return StructuralFacts{}, breach
	}
	facts := structuralFactsOf(imported)
	for _, breachIn := range boundsOfStructuralFacts {
		if breach := breachIn(facts, limits); breach != nil {
			return StructuralFacts{}, breach
		}
	}
	return facts, nil
}

func (facts StructuralFacts) moduleGraphDelta() ModuleGraphDelta {
	return ModuleGraphDelta{
		EdgesAdded:   slices.Clone(facts.edgesAdded),
		EdgesRemoved: slices.Clone(facts.edgesRemoved),
		Incomplete:   importResolutionIn(facts.imports).unresolved > 0,
	}
}

func changeImportCountBreachIn(imported []ImportedFile, limits Limits) error {
	published := headImportCountOf(imported)
	if published > limits.MaxImportsPerChange {
		return fmt.Errorf("%w: %d imports exceeds %d",
			ErrTooManyImportsInChange, published, limits.MaxImportsPerChange)
	}
	return nil
}

func structuralFactsOf(imported []ImportedFile) StructuralFacts {
	return StructuralFacts{
		imports:      importsOf(imported),
		edgesAdded:   edgesIn(imported, ImportedFile.edgesAdded),
		edgesRemoved: edgesIn(imported, ImportedFile.edgesRemoved),
	}
}

func edgeCountBreachIn(facts StructuralFacts, limits Limits) error {
	edges := len(facts.edgesAdded) + len(facts.edgesRemoved)
	if edges > limits.MaxGraphEdges {
		return fmt.Errorf("%w: %d edges exceeds %d", ErrTooManyEdges, edges, limits.MaxGraphEdges)
	}
	return nil
}

func serialisedBytesBreachIn(facts StructuralFacts, limits Limits) error {
	serialised, err := facts.serialisedLength()
	if err != nil {
		return err
	}
	if serialised > limits.MaxStructuralBytes {
		return fmt.Errorf("%w: %d bytes exceeds %d",
			ErrStructuralFactsTooLarge, serialised, limits.MaxStructuralBytes)
	}
	return nil
}

func (facts StructuralFacts) serialisedLength() (int, error) {
	imports, err := json.Marshal(facts.imports)
	if err != nil {
		return 0, fmt.Errorf("serialise the imports: %w", err)
	}
	delta, err := json.Marshal(facts.moduleGraphDelta())
	if err != nil {
		return 0, fmt.Errorf("serialise the module graph delta: %w", err)
	}
	return len(imports) + len(delta), nil
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
