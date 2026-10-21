// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "errors"

type ImportExtractor interface {
	Claims(path string) bool
	ImportsIn(source BoundedSource) ([]string, error)
}

var ErrUnparsableSource = errors.New("the source does not parse")
