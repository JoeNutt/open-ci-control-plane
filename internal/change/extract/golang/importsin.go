// SPDX-License-Identifier: GPL-3.0-or-later

package golang

func ImportsIn(source []byte) ([]string, error) {
	return []string{
		"context",
		"embed",
		"github.com/JoeNutt/open-ci-control-plane/internal/change/domain",
	}, nil
}
