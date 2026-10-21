// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"slices"
)

var ErrSourceTooLarge = errors.New("source exceeds the source byte bound")

type BoundedSource struct{ contents []byte }

func ParseBoundedSource(contents []byte, limits Limits) (BoundedSource, error) {
	if len(contents) > limits.MaxSourceBytes {
		return BoundedSource{}, fmt.Errorf("%w: %d bytes exceeds %d",
			ErrSourceTooLarge, len(contents), limits.MaxSourceBytes)
	}
	return BoundedSource{contents: slices.Clone(contents)}, nil
}

func (source BoundedSource) Reader() io.Reader { return bytes.NewReader(source.contents) }
