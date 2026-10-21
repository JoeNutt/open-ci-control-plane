// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type ImportExtractor interface {
	Claims(path string) bool
	ImportsIn(source BoundedSource) ([]string, error)
}
