// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "errors"

var ErrSourceTooLarge = errors.New("source exceeds the source byte bound")

type BoundedSource struct{}

func ParseBoundedSource(contents []byte, limits Limits) (BoundedSource, error) {
	return BoundedSource{}, ErrSourceTooLarge
}
