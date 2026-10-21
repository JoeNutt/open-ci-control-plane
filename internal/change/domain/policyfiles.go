// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"slices"
)

const MaxPolicyFiles = 256

var ErrTooManyPolicyFiles = errors.New("policy repository exceeds the policy file bound")

type PolicyFile struct {
	Path    string
	Content []byte
}

type BoundedPolicyFiles struct {
	files []PolicyFile
}

func ParseBoundedPolicyFiles(files []PolicyFile) (BoundedPolicyFiles, error) {
	if len(files) > MaxPolicyFiles {
		return BoundedPolicyFiles{}, fmt.Errorf("%w: %d exceeds %d", ErrTooManyPolicyFiles, len(files), MaxPolicyFiles)
	}
	return BoundedPolicyFiles{files: slices.Clone(files)}, nil
}
