// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"slices"
)

func parseDeclaration[Declaration ~string](
	name string, declared []Declaration, unknown error,
) (Declaration, error) {
	stated := Declaration(name)
	if slices.Contains(declared, stated) {
		return stated, nil
	}
	return "", fmt.Errorf("%w: %q", unknown, name)
}
