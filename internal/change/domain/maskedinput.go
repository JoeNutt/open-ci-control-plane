// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

var ErrUnmaskableField = errors.New("field cannot be masked")

type MaskedInput struct {
	document Document
	redacted []string
}

func MaskedInputOf(document Document) (MaskedInput, error) {
	fields, err := sensitiveFieldsOf(document.Diff.Sensitive)
	if err != nil {
		return MaskedInput{}, err
	}
	masked := copyForMasking(document)
	redacted := redactAll(&masked, fields)
	return MaskedInput{document: masked, redacted: redacted}, nil
}

func (masked MaskedInput) MarshalJSON() ([]byte, error) { return json.Marshal(masked.document) }

func (masked MaskedInput) RedactedPointers() []string { return slices.Clone(masked.redacted) }

const arrayIndexSlot = "{index}"

type maskableField struct {
	shape  []string
	redact func(document *Document, index int) bool
}

var maskableFields = []maskableField{
	{shape: []string{"pr", "author", "login"}, redact: redactAuthorLogin},
	{shape: []string{"pr", "approvals", arrayIndexSlot, "login"}, redact: redactApprovalLogin},
	{shape: []string{"analyzers", "findings", arrayIndexSlot}, redact: redactAnalyzerFinding},
}

type sensitiveField struct {
	stated string
	index  int
	redact func(document *Document, index int) bool
}

func sensitiveFieldsOf(sensitive []string) ([]sensitiveField, error) {
	fields := make([]sensitiveField, 0, len(sensitive))
	for _, stated := range sensitive {
		field, err := parseSensitiveField(stated)
		if err != nil {
			return nil, fmt.Errorf("mask %q: %w", stated, err)
		}
		fields = append(fields, field)
	}
	return fields, nil
}

func parseSensitiveField(stated string) (sensitiveField, error) {
	pointer, err := ParseJSONPointer(stated)
	if err != nil {
		return sensitiveField{}, err
	}
	tokens := pointer.Tokens()
	for _, maskable := range maskableFields {
		index, matches, err := maskable.indexIn(tokens)
		if err != nil {
			return sensitiveField{}, err
		}
		if matches {
			return sensitiveField{stated: stated, index: index, redact: maskable.redact}, nil
		}
	}
	return sensitiveField{}, ErrUnmaskableField
}

func (maskable maskableField) indexIn(tokens []string) (int, bool, error) {
	if !maskable.matchesMembers(tokens) {
		return 0, false, nil
	}
	slot := slices.Index(maskable.shape, arrayIndexSlot)
	if slot < 0 {
		return 0, true, nil
	}
	index, err := parseArrayIndex(tokens[slot])
	if err != nil {
		return 0, false, err
	}
	return index, true, nil
}

func (maskable maskableField) matchesMembers(tokens []string) bool {
	if len(tokens) != len(maskable.shape) {
		return false
	}
	for at, expected := range maskable.shape {
		if expected != arrayIndexSlot && tokens[at] != expected {
			return false
		}
	}
	return true
}

func parseArrayIndex(token string) (int, error) {
	if hasForbiddenIndexSpelling(token) {
		return 0, fmt.Errorf("%w: %q is not an array index", ErrMalformedJSONPointer, token)
	}
	index, err := strconv.Atoi(token)
	if err != nil {
		return 0, fmt.Errorf("%w: %q is not an array index", ErrUnmaskableField, token)
	}
	return index, nil
}

func hasForbiddenIndexSpelling(token string) bool {
	if strings.HasPrefix(token, "+") {
		return true
	}
	isLongerThanOneCharacter := len(token) > 1
	hasLeadingZero := isLongerThanOneCharacter && strings.HasPrefix(token, "0")
	hasMinusSign := isLongerThanOneCharacter && strings.HasPrefix(token, "-")
	return hasLeadingZero || hasMinusSign
}

func copyForMasking(document Document) Document {
	masked := document
	masked.PR.Approvals = slices.Clone(document.PR.Approvals)
	masked.Analyzers.Findings = slices.Clone(document.Analyzers.Findings)
	return masked
}

func redactAll(document *Document, fields []sensitiveField) []string {
	removed := make(map[string]bool, len(fields))
	for _, field := range highestIndexFirst(fields) {
		removed[field.stated] = field.redact(document, field.index)
	}
	var redacted []string
	for _, field := range fields {
		if removed[field.stated] {
			redacted = append(redacted, field.stated)
		}
	}
	return redacted
}

func highestIndexFirst(fields []sensitiveField) []sensitiveField {
	ordered := slices.Clone(fields)
	slices.SortStableFunc(ordered, func(first, second sensitiveField) int {
		return cmp.Compare(second.index, first.index)
	})
	return ordered
}

func redactAuthorLogin(document *Document, _ int) bool {
	if document.PR.Author.Login == "" {
		return false
	}
	document.PR.Author.Login = ""
	return true
}

func redactApprovalLogin(document *Document, index int) bool {
	if index >= len(document.PR.Approvals) || document.PR.Approvals[index].Login == "" {
		return false
	}
	document.PR.Approvals[index].Login = ""
	return true
}

func redactAnalyzerFinding(document *Document, index int) bool {
	if index >= len(document.Analyzers.Findings) {
		return false
	}
	document.Analyzers.Findings = slices.Delete(document.Analyzers.Findings, index, index+1)
	return true
}
