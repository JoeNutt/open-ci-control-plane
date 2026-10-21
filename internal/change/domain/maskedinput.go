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

var redactorByPointer = map[string]func(document *Document) bool{
	"/pr/author/login": redactAuthorLogin,
}

type maskableArrayElement struct {
	prefix string
	suffix string
	redact func(document *Document, index int) bool
}

var maskableArrayElements = []maskableArrayElement{
	{prefix: "/pr/approvals/", suffix: "/login", redact: redactApprovalLogin},
	{prefix: "/analyzers/findings/", suffix: "", redact: redactAnalyzerFinding},
}

type sensitiveField struct {
	stated string
	index  int
	redact func(document *Document) bool
}

func sensitiveFieldsOf(sensitive []string) ([]sensitiveField, error) {
	distinct := slices.Compact(slices.Sorted(slices.Values(sensitive)))
	fields := make([]sensitiveField, 0, len(distinct))
	for _, stated := range distinct {
		field, err := parseSensitiveField(stated)
		if err != nil {
			return nil, err
		}
		fields = append(fields, field)
	}
	return fields, nil
}

func parseSensitiveField(stated string) (sensitiveField, error) {
	if redact, isMaskable := redactorByPointer[stated]; isMaskable {
		return sensitiveField{stated: stated, redact: redact}, nil
	}
	for _, element := range maskableArrayElements {
		if index, matches := element.indexIn(stated); matches {
			redact := func(document *Document) bool { return element.redact(document, index) }
			return sensitiveField{stated: stated, index: index, redact: redact}, nil
		}
	}
	return sensitiveField{}, fmt.Errorf("%w: %q", ErrUnmaskableField, stated)
}

func (element maskableArrayElement) indexIn(stated string) (int, bool) {
	afterPrefix, hasPrefix := strings.CutPrefix(stated, element.prefix)
	token, hasSuffix := strings.CutSuffix(afterPrefix, element.suffix)
	if !hasPrefix || !hasSuffix {
		return 0, false
	}
	return arrayIndexOf(token)
}

func arrayIndexOf(token string) (int, bool) {
	index, err := strconv.ParseUint(token, 10, strconv.IntSize-1)
	if err != nil || strconv.FormatUint(index, 10) != token {
		return 0, false
	}
	return int(index), true
}

func copyForMasking(document Document) Document {
	masked := document
	masked.PR.Approvals = slices.Clone(document.PR.Approvals)
	masked.Analyzers.Findings = slices.Clone(document.Analyzers.Findings)
	return masked
}

func redactAll(document *Document, fields []sensitiveField) []string {
	var redacted []string
	for _, field := range highestIndexFirst(fields) {
		if field.redact(document) {
			redacted = append(redacted, field.stated)
		}
	}
	slices.Sort(redacted)
	return redacted
}

func highestIndexFirst(fields []sensitiveField) []sensitiveField {
	ordered := slices.Clone(fields)
	slices.SortStableFunc(ordered, func(first, second sensitiveField) int {
		return cmp.Compare(second.index, first.index)
	})
	return ordered
}

func redactAuthorLogin(document *Document) bool {
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
