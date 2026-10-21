// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"context"
	"errors"
)

var ErrNoSuchFile = errors.New("the forge reports no such file at that commit")

type FileAtCommit struct {
	Repository string
	Commit     CommitSHA
	Path       string
}

type FileContents interface {
	ContentsAt(ctx context.Context, file FileAtCommit) ([]byte, error)
}
