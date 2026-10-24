// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	TopLevelTestPrefix = "Test"
	identifierJoiner   = '_'
)

var errNotATopLevelTest = errors.New("the name is not a top-level Go test function")

type TopLevelTestName struct{ name string }

func ParseTopLevelTestName(stated string) (TopLevelTestName, error) {
	if !isIdentifierInAnyScript(stated) || !IsTopLevelTestName(stated) {
		return TopLevelTestName{}, fmt.Errorf("%w: %d bytes", errNotATopLevelTest, len(stated))
	}
	return TopLevelTestName{name: stated}, nil
}

func (name TopLevelTestName) compare(other TopLevelTestName) int {
	return strings.Compare(name.name, other.name)
}

func IsTopLevelTestName(name string) bool {
	rest, hasPrefix := strings.CutPrefix(name, TopLevelTestPrefix)
	if !hasPrefix {
		return false
	}
	next, _ := utf8.DecodeRuneInString(rest)
	return rest == "" || !unicode.IsLower(next)
}

func isIdentifierInAnyScript(name string) bool {
	return strings.IndexFunc(name, isOutsideIdentifiers) < 0
}

func isOutsideIdentifiers(character rune) bool {
	return !unicode.IsLetter(character) && !unicode.IsDigit(character) && character != identifierJoiner
}
