// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
)

// SchemaMajor is a major version of the change document contract. Fields are added within
// a major version and never repurposed or renamed; a removal is a new major version.
type SchemaMajor int

// CurrentSchemaMajor is the change document major version this release publishes.
const CurrentSchemaMajor SchemaMajor = 1

// ErrSchemaMajorMismatch reports a bundle written against another major version of the
// change document.
var ErrSchemaMajorMismatch = errors.New("bundle declares another change document major version")

// AcceptBundleMajor converts the major version a bundle declares into the SchemaMajor it is
// evaluated under. It returns ErrSchemaMajorMismatch, naming both versions, when the bundle
// was written against a major version this release does not publish.
func AcceptBundleMajor(declared int) (SchemaMajor, error) {
	if declared != int(CurrentSchemaMajor) {
		return 0, fmt.Errorf("%w: bundle declares %d, this release publishes %d",
			ErrSchemaMajorMismatch, declared, CurrentSchemaMajor)
	}
	return CurrentSchemaMajor, nil
}
