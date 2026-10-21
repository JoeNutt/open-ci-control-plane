// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"net/http"
	"slices"
	"strings"
)

const (
	eventHeader    = "X-GitHub-Event"
	deliveryHeader = "X-GitHub-Delivery"
)

const (
	lowerASCIILetters  = "abcdefghijklmnopqrstuvwxyz"
	upperASCIILetters  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	asciiDigits        = "0123456789"
	eventCharacters    = lowerASCIILetters + "_"
	deliveryCharacters = lowerASCIILetters + upperASCIILetters + asciiDigits + "-"
)

const (
	maxEventBytes    = 100
	maxDeliveryBytes = 200
)

type headerRule struct {
	name       string
	maxBytes   int
	characters string
}

var requiredHeaders = []headerRule{
	{name: eventHeader, maxBytes: maxEventBytes, characters: eventCharacters},
	{name: deliveryHeader, maxBytes: maxDeliveryBytes, characters: deliveryCharacters},
}

func headerProblemIn(stated http.Header) (fieldProblem, bool) {
	if absent := slices.IndexFunc(requiredHeaders, isAbsentFrom(stated)); absent >= 0 {
		return missingHeaderProblem(requiredHeaders[absent].name), true
	}
	if malformed := slices.IndexFunc(requiredHeaders, isMalformedIn(stated)); malformed >= 0 {
		return malformedHeaderProblem(requiredHeaders[malformed].name), true
	}
	return fieldProblem{}, false
}

func isAbsentFrom(stated http.Header) func(headerRule) bool {
	return func(rule headerRule) bool { return stated.Get(rule.name) == "" }
}

func isMalformedIn(stated http.Header) func(headerRule) bool {
	return func(rule headerRule) bool {
		value := stated.Get(rule.name)
		return len(value) > rule.maxBytes || strings.Trim(value, rule.characters) != ""
	}
}
