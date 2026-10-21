// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
)

var ErrSourceTooLarge = errors.New("source exceeds the source byte bound")

type BoundedSource struct{ contents []byte }

func ParseBoundedSource(contents []byte, limits Limits) (BoundedSource, error) {
	if len(contents) > limits.MaxSourceBytes {
		return BoundedSource{}, fmt.Errorf("%w: %d bytes exceeds %d",
			ErrSourceTooLarge, len(contents), limits.MaxSourceBytes)
	}
	return BoundedSource{contents: contents}, nil
}

func (source BoundedSource) Contents() []byte { return source.contents }
