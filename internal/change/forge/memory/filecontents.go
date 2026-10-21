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
	recorded, held := contents.Files[file]
	if !held {
		return nil, domain.ErrNoSuchFile
	}
	return slices.Clone(recorded), nil
}
