// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"encoding/json"
	"strconv"
	"strings"
)

type MaskedInput struct {
	tree     map[string]any
	redacted []string
}

func MaskedInputOf(document Document) (MaskedInput, error) {
	tree, err := treeOf(document)
	if err != nil {
		return MaskedInput{}, err
	}
	var redacted []string
	for _, stated := range document.Diff.Sensitive {
		removed, err := removeNamed(tree, stated)
		if err != nil {
			return MaskedInput{}, err
		}
		if removed {
			redacted = append(redacted, stated)
		}
	}
	return MaskedInput{tree: sweep(tree).(map[string]any), redacted: redacted}, nil
}

type removedElement struct{}

func sweep(node any) any {
	switch container := node.(type) {
	case map[string]any:
		for key, child := range container {
			container[key] = sweep(child)
		}
		return container
	case []any:
		kept := []any{}
		for _, child := range container {
			if _, removed := child.(removedElement); removed {
				continue
			}
			kept = append(kept, sweep(child))
		}
		return kept
	default:
		return node
	}
}

func (masked MaskedInput) MarshalJSON() ([]byte, error) { return json.Marshal(masked.tree) }

func (masked MaskedInput) RedactedPointers() []string { return masked.redacted }

func treeOf(document Document) (map[string]any, error) {
	encoded, err := json.Marshal(document)
	if err != nil {
		return nil, err
	}
	var tree map[string]any
	if err := json.Unmarshal(encoded, &tree); err != nil {
		return nil, err
	}
	return tree, nil
}

func removeNamed(tree map[string]any, stated string) (bool, error) {
	pointer, err := ParseJSONPointer(stated)
	if err != nil {
		return false, err
	}
	return removeAt(tree, pointer)
}

func removeAt(tree map[string]any, pointer JSONPointer) (bool, error) {
	tokens := pointer.Tokens()
	var node any = tree
	for _, token := range tokens[:len(tokens)-1] {
		child, exists, err := childOf(node, token)
		if err != nil {
			return false, err
		}
		if !exists {
			return false, nil
		}
		node = child
	}
	return removeChild(node, tokens[len(tokens)-1])
}

func childOf(node any, token string) (any, bool, error) {
	switch container := node.(type) {
	case map[string]any:
		child, exists := container[token]
		return child, exists, nil
	case []any:
		index, exists, err := arrayIndex(container, token)
		if !exists {
			return nil, false, err
		}
		return container[index], true, nil
	default:
		return nil, false, nil
	}
}

func removeChild(node any, token string) (bool, error) {
	switch container := node.(type) {
	case map[string]any:
		_, exists := container[token]
		delete(container, token)
		return exists, nil
	case []any:
		index, exists, err := arrayIndex(container, token)
		if !exists {
			return false, err
		}
		container[index] = removedElement{}
		return true, nil
	}
	return false, nil
}

func arrayIndex(elements []any, token string) (int, bool, error) {
	if hasForbiddenIndexSpelling(token) {
		return 0, false, ErrMalformedJSONPointer
	}
	index, err := strconv.Atoi(token)
	if err != nil || index >= len(elements) {
		return 0, false, nil
	}
	return index, true, nil
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
