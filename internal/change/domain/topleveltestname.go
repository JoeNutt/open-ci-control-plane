// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const TopLevelTestPrefix = "Test"

var errNotATopLevelTest = errors.New("the name is not a top-level Go test function")

type TopLevelTestName struct{ name string }

func ParseTopLevelTestName(stated string) (TopLevelTestName, error) {
	rest, isTest := strings.CutPrefix(stated, TopLevelTestPrefix)
	if !isTest || strings.IndexFunc(stated, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_'
	}) >= 0 {
		return TopLevelTestName{}, fmt.Errorf("%w: %q", errNotATopLevelTest, stated)
	}
	if rest != "" {
		next, _ := utf8.DecodeRuneInString(rest)
		if unicode.IsLower(next) {
			return TopLevelTestName{}, fmt.Errorf("%w: %q", errNotATopLevelTest, stated)
		}
	}
	return TopLevelTestName{name: stated}, nil
}

func (name TopLevelTestName) compare(other TopLevelTestName) int {
	return strings.Compare(name.name, other.name)
}
