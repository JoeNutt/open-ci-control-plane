// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"strings"
)

const TopLevelTestPrefix = "Test"

var errNotATopLevelTest = errors.New("the name is not a top-level Go test function")

type TopLevelTestName struct{ name string }

func ParseTopLevelTestName(stated string) (TopLevelTestName, error) {
	rest, isTest := strings.CutPrefix(stated, TopLevelTestPrefix)
	if !isTest || strings.Trim(rest, identifierCharacters) != "" {
		return TopLevelTestName{}, fmt.Errorf("%w: %q", errNotATopLevelTest, stated)
	}
	return TopLevelTestName{name: stated}, nil
}

func (name TopLevelTestName) compare(other TopLevelTestName) int {
	return strings.Compare(name.name, other.name)
}
