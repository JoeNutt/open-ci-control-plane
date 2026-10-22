// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "fmt"

type BranchTips struct {
	base CommitSHA
	head CommitSHA
}

func ParseBranchTips(base, head CommitSHA) (BranchTips, error) {
	if base == (CommitSHA{}) || head == (CommitSHA{}) {
		return BranchTips{}, fmt.Errorf("%w: a branch tip names no commit", ErrMalformedCommitSHA)
	}
	return BranchTips{base: base, head: head}, nil
}

func (tips BranchTips) Base() CommitSHA { return tips.base }

func (tips BranchTips) Head() CommitSHA { return tips.head }
