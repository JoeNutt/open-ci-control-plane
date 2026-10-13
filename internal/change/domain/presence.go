// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"encoding/json"
	"slices"
)

type Declared[Element any] []Element

func (declared Declared[Element]) MarshalJSON() ([]byte, error) {
	if len(declared) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal([]Element(declared))
}

type Paths []string

func (paths Paths) MarshalJSON() ([]byte, error) {
	sorted := slices.Clone(paths)
	slices.Sort(sorted)
	return Declared[string](sorted).MarshalJSON()
}
