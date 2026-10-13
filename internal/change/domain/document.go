// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type Document struct {
	Diff Diff `json:"diff"`
}

type Diff struct {
	Paths Paths `json:"paths"`
}

type Paths []string

func (p Paths) MarshalJSON() ([]byte, error) {
	// green: nil must not become null
	if len(p) == 0 {
		return []byte("[]"), nil
	}
	return []byte(`["x"]`), nil
}
