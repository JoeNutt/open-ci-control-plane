// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
)

const commitSHAHexDigits = 40

var ErrMalformedCommitSHA = errors.New("commit SHA is not a full lower-case object name")

type CommitSHA struct {
	value string
}

func ParseCommitSHA(stated string) (CommitSHA, error) {
	if len(stated) != commitSHAHexDigits || !isLowerHex(stated) {
		return CommitSHA{}, fmt.Errorf("%w: %q", ErrMalformedCommitSHA, stated)
	}
	return CommitSHA{value: stated}, nil
}

func (sha CommitSHA) String() string { return sha.value }
