// SPDX-License-Identifier: GPL-3.0-or-later

package memory

import (
	"context"
	"slices"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type FileContents struct {
	Files map[domain.FileAtCommit][]byte
}

func (contents FileContents) ContentsAt(
	_ context.Context, file domain.FileAtCommit,
) ([]byte, error) {
	return slices.Clone(contents.Files[file]), nil
}
