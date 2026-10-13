// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"encoding/json"
	"slices"
)

type Declared[E any] []E

func (d Declared[E]) MarshalJSON() ([]byte, error) {
	if len(d) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal([]E(d))
}

type Paths []string

func (p Paths) MarshalJSON() ([]byte, error) {
	sorted := slices.Clone(p)
	slices.Sort(sorted)
	return Declared[string](sorted).MarshalJSON()
}
