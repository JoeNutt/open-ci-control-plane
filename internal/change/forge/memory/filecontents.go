// SPDX-License-Identifier: GPL-3.0-or-later

package memory

import (
	"context"
	"fmt"
	"slices"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type FileContents struct {
	Files map[domain.FileAtCommit][]byte
}

func (contents FileContents) ContentsAt(
	_ context.Context, file domain.FileAtCommit,
) ([]byte, error) {
	answer, recorded := contents.Files[file]
	if !recorded {
		return nil, fmt.Errorf("%w: %s in %s at %s",
			domain.ErrNoSuchFile, file.Path, file.Repository, file.Commit)
	}
	return slices.Clone(answer), nil
}
