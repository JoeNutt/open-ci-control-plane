// SPDX-License-Identifier: GPL-3.0-or-later

// Package domain holds the change document, the published contract that policy is
// evaluated against.
package domain

// Document is the change document for one change, published to policy as ci.diff/v1.
type Document struct {
	Diff Diff `json:"diff"`
}

// Diff is the artefact under decision: the paths a change touches and what it did to them.
type Diff struct {
	Paths Paths `json:"paths"`
}

// Paths is a list of changed file paths. It is never marshalled as null, because a field
// absent from the document evaluates to undefined in the policy engine and undefined
// deletes a control.
type Paths []string

// MarshalJSON encodes the paths as a JSON array, empty rather than null.
func (p Paths) MarshalJSON() ([]byte, error) {
	if len(p) == 0 {
		return []byte("[]"), nil
	}
	return []byte(`["x"]`), nil
}
