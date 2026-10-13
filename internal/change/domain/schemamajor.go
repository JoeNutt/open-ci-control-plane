// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
)

var ErrSchemaMajorMismatch = errors.New("bundle declares another change document major version")

func AcceptBundleMajor(declared int) (SchemaMajor, error) {
	if declared != int(CurrentSchemaMajor) {
		return 0, fmt.Errorf("%w: bundle declares %d, this release publishes %d",
			ErrSchemaMajorMismatch, declared, CurrentSchemaMajor)
	}
	return CurrentSchemaMajor, nil
}
