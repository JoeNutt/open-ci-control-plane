// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"errors"
	"fmt"
	"strings"
)

const (
	commitSHADigits = 40
	lowerHexDigits  = "0123456789abcdef"
)

var ErrMalformedHeadSHA = errors.New("the head commit is not a 40-digit lowercase hex SHA")

type commitSHA string

func parseCommitSHA(stated string) (commitSHA, error) {
	if len(stated) != commitSHADigits || !isLowerHex(stated) {
		return "", fmt.Errorf("%w: %q", ErrMalformedHeadSHA, stated)
	}
	return commitSHA(stated), nil
}

func isLowerHex(digits string) bool {
	return strings.Trim(digits, lowerHexDigits) == ""
}
