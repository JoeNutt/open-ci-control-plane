// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
)

var ErrMissingBundleRevision = errors.New("decision carries no bundle revision")

type BundleRevision struct {
	commit CommitSHA
}

func ParseBundleRevision(stated string) (BundleRevision, error) {
	if stated == "" {
		return BundleRevision{}, ErrMissingBundleRevision
	}
	commit, err := ParseCommitSHA(stated)
	if err != nil {
		return BundleRevision{}, fmt.Errorf("bundle revision: %w", err)
	}
	return BundleRevision{commit: commit}, nil
}

func (revision BundleRevision) String() string { return revision.commit.String() }

func (revision BundleRevision) IsKnown() bool { return revision != BundleRevision{} }
