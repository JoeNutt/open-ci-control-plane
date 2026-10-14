// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "strings"

const (
	repositorySeparator = "/"
	asciiAlphanumerics  = "abcdefghijklmnopqrstuvwxyz" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"0123456789"
	maxRepositoryLength = maxOwnerBytes + len(repositorySeparator) + maxNameBytes
	maxOwnerBytes       = 39
	maxNameBytes        = 100
)

type segmentRule struct {
	maxBytes          int
	firstCharacters   string
	wholeCharacters   string
	requiresOneBeyond string
}

var ownerRule = segmentRule{
	maxBytes:        maxOwnerBytes,
	firstCharacters: asciiAlphanumerics,
	wholeCharacters: asciiAlphanumerics + "-",
}

var nameRule = segmentRule{
	maxBytes:          maxNameBytes,
	firstCharacters:   asciiAlphanumerics + ".",
	wholeCharacters:   asciiAlphanumerics + "-._",
	requiresOneBeyond: ".",
}

func (rule segmentRule) admits(segment string) bool {
	if len(segment) == 0 || len(segment) > rule.maxBytes {
		return false
	}
	if strings.IndexByte(rule.firstCharacters, segment[0]) < 0 {
		return false
	}
	if strings.Trim(segment, rule.requiresOneBeyond) == "" {
		return false
	}
	return strings.Trim(segment, rule.wholeCharacters) == ""
}
