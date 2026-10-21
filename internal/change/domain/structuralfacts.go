// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "errors"

var ErrTooManyImportsInChange = errors.New("change exceeds the per-change import bound")

type StructuralFacts struct{}

func ParseStructuralFacts(imported []ImportedFile, limits Limits) (StructuralFacts, error) {
	return StructuralFacts{}, ErrTooManyImportsInChange
}
