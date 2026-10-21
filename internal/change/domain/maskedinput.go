// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"encoding/json"
	"strconv"
	"strings"
)

type MaskedInput struct {
	tree map[string]any
}

func MaskedInputOf(document Document) (MaskedInput, error) {
	tree, err := treeOf(document)
	if err != nil {
		return MaskedInput{}, err
	}
	for _, stated := range document.Diff.Sensitive {
		pointer, err := ParseJSONPointer(stated)
		if err != nil {
			return MaskedInput{}, err
		}
		if err := removeAt(tree, pointer); err != nil {
			return MaskedInput{}, err
		}
	}
	return MaskedInput{tree: sweep(tree).(map[string]any)}, nil
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

func removeAt(tree map[string]any, pointer JSONPointer) error {
	tokens := pointer.Tokens()
	var node any = tree
	for _, token := range tokens[:len(tokens)-1] {
		child, exists, err := childOf(node, token)
		if err != nil {
			return err
		}
		if !exists {
			return nil
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

func removeChild(node any, token string) error {
	switch container := node.(type) {
	case map[string]any:
		delete(container, token)
	case []any:
		index, exists, err := arrayIndex(container, token)
		if !exists {
			return err
		}
		container[index] = removedElement{}
	}
	return nil
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
