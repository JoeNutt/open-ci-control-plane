// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"encoding/json"
	"slices"
)

// Declared is a list of values of one declared document field. It marshals as an empty
// array, never as null.
type Declared[E any] []E

func (d Declared[E]) MarshalJSON() ([]byte, error) {
	if len(d) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal([]E(d))
}

// Paths is a declared list of repository-relative file paths. It marshals lexicographically
// sorted.
type Paths []string

func (p Paths) MarshalJSON() ([]byte, error) {
	sorted := slices.Clone(p)
	slices.Sort(sorted)
	return Declared[string](sorted).MarshalJSON()
}
