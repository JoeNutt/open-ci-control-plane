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
	var fields []sensitiveField
	for _, stated := range document.Diff.Sensitive {
		field, err := parseSensitiveField(stated)
		if err != nil {
			return MaskedInput{}, fmt.Errorf("mask %q: %w", stated, err)
		}
		fields = append(fields, field)
	}

	masked := document
	masked.PR.Approvals = slices.Clone(document.PR.Approvals)
	masked.Analyzers.Findings = slices.Clone(document.Analyzers.Findings)

	order := make([]int, len(fields))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int { return cmp.Compare(fields[b].index, fields[a].index) })
	removed := make([]bool, len(fields))
	for _, i := range order {
		removed[i] = fields[i].redactIn(&masked)
	}

	var redacted []string
	for i, field := range fields {
		if removed[i] {
			redacted = append(redacted, field.stated)
		}
	}
	return MaskedInput{document: masked, redacted: redacted}, nil
}

func (masked MaskedInput) MarshalJSON() ([]byte, error) { return json.Marshal(masked.document) }

func (masked MaskedInput) RedactedPointers() []string { return slices.Clone(masked.redacted) }

type sensitiveLocation int

const (
	authorLogin sensitiveLocation = iota
	approvalLogin
	analyzerFinding
)

type sensitiveField struct {
	stated   string
	location sensitiveLocation
	index    int
}

func parseSensitiveField(stated string) (sensitiveField, error) {
	pointer, err := ParseJSONPointer(stated)
	if err != nil {
		return sensitiveField{}, err
	}
	tokens := pointer.Tokens()
	switch {
	case slices.Equal(tokens, []string{"pr", "author", "login"}):
		return sensitiveField{stated: stated, location: authorLogin}, nil
	case len(tokens) == 4 && tokens[0] == "pr" && tokens[1] == "approvals" && tokens[3] == "login":
		index, err := parseIndex(tokens[2])
		if err != nil {
			return sensitiveField{}, err
		}
		return sensitiveField{stated: stated, location: approvalLogin, index: index}, nil
	case len(tokens) == 3 && tokens[0] == "analyzers" && tokens[1] == "findings":
		index, err := parseIndex(tokens[2])
		if err != nil {
			return sensitiveField{}, err
		}
		return sensitiveField{stated: stated, location: analyzerFinding, index: index}, nil
	}
	return sensitiveField{}, ErrUnmaskableField
}

func parseIndex(token string) (int, error) {
	if strings.HasPrefix(token, "+") || len(token) > 1 && (strings.HasPrefix(token, "0") || strings.HasPrefix(token, "-")) {
		return 0, fmt.Errorf("%w: %q is not an array index", ErrMalformedJSONPointer, token)
	}
	index, err := strconv.Atoi(token)
	if err != nil {
		return 0, ErrUnmaskableField
	}
	return index, nil
}

func (field sensitiveField) redactIn(document *Document) bool {
	switch field.location {
	case authorLogin:
		if document.PR.Author.Login == "" {
			return false
		}
		document.PR.Author.Login = ""
		return true
	case approvalLogin:
		if field.index >= len(document.PR.Approvals) || document.PR.Approvals[field.index].Login == "" {
			return false
		}
		document.PR.Approvals[field.index].Login = ""
		return true
	case analyzerFinding:
		if field.index >= len(document.Analyzers.Findings) {
			return false
		}
		document.Analyzers.Findings = slices.Delete(document.Analyzers.Findings, field.index, field.index+1)
		return true
	}
	return false
}
