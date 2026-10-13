// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"encoding/json"
	"slices"
)

// Declared is a list of values of one declared document field. It marshals as an empty
// array rather than as null, because a field absent from the document evaluates to
// undefined in the policy engine and undefined deletes a control.
type Declared[E any] []E

// MarshalJSON encodes the list as a JSON array, empty rather than null.
func (d Declared[E]) MarshalJSON() ([]byte, error) {
	if len(d) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal([]E(d))
}

// Paths is a declared list of repository-relative file paths. It marshals lexicographically
// sorted, so a policy that compares or hashes it reads one order for one change.
type Paths []string

// MarshalJSON encodes the paths as a sorted JSON array, empty rather than null.
func (p Paths) MarshalJSON() ([]byte, error) {
	sorted := slices.Clone(p)
	slices.Sort(sorted)
	return Declared[string](sorted).MarshalJSON()
}
