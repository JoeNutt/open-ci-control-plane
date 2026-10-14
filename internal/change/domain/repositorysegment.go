// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "strings"

const (
	repositorySeparator   = "/"
	maxRepositorySegment  = 100
	maxRepositoryLength   = 2*maxRepositorySegment + len(repositorySeparator)
	repositoryPunctuation = "._-"
)

func repositorySegmentOK(segment string) bool {
	if len(segment) == 0 || len(segment) > maxRepositorySegment {
		return false
	}
	if !asciiAlphanumeric(segment[0]) {
		return false
	}
	for index := 1; index < len(segment); index++ {
		if !repositoryCharacterOK(segment[index]) {
			return false
		}
	}
	return true
}

func repositoryCharacterOK(character byte) bool {
	return asciiAlphanumeric(character) ||
		strings.IndexByte(repositoryPunctuation, character) >= 0
}

func asciiAlphanumeric(character byte) bool {
	return 'a' <= character && character <= 'z' ||
		'A' <= character && character <= 'Z' ||
		'0' <= character && character <= '9'
}

func nameSegmentOK(segment string) bool {
	if len(segment) == 0 || len(segment) > maxRepositorySegment {
		return false
	}
	if strings.Trim(segment, ".") == "" {
		return false
	}
	if !asciiAlphanumeric(segment[0]) && segment[0] != '.' {
		return false
	}
	for index := 1; index < len(segment); index++ {
		if !repositoryCharacterOK(segment[index]) {
			return false
		}
	}
	return true
}

func ownerSegmentOK(segment string) bool {
	if len(segment) == 0 || len(segment) > maxRepositorySegment {
		return false
	}
	if !asciiAlphanumeric(segment[0]) {
		return false
	}
	for index := 1; index < len(segment); index++ {
		if !asciiAlphanumeric(segment[index]) && segment[index] != '-' {
			return false
		}
	}
	return true
}
