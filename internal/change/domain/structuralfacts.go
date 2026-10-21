// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
)

var ErrTooManyImportsInChange = errors.New("change exceeds the per-change import bound")

type StructuralFacts struct{ imported []ImportedFile }

func ParseStructuralFacts(imported []ImportedFile, limits Limits) (StructuralFacts, error) {
	published := headImportCountOf(imported)
	if published > limits.MaxImportsPerChange {
		return StructuralFacts{}, fmt.Errorf("%w: %d imports exceeds %d",
			ErrTooManyImportsInChange, published, limits.MaxImportsPerChange)
	}
	return StructuralFacts{imported: imported}, nil
}

func headImportCountOf(imported []ImportedFile) int {
	count := 0
	for _, file := range imported {
		count += len(file.Head.imports)
	}
	return count
}
