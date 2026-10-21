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
	return MaskedInput{tree: tree}, nil
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
	parent, isObject := node.(map[string]any)
	if !isObject {
		return nil
	}
	delete(parent, tokens[len(tokens)-1])
	return nil
}

func childOf(node any, token string) (any, bool, error) {
	switch container := node.(type) {
	case map[string]any:
		child, exists := container[token]
		return child, exists, nil
	case []any:
		return arrayElement(container, token)
	default:
		return nil, false, nil
	}
}

func arrayElement(elements []any, token string) (any, bool, error) {
	if hasForbiddenIndexSpelling(token) {
		return nil, false, ErrMalformedJSONPointer
	}
	index, err := strconv.Atoi(token)
	if err != nil || index >= len(elements) {
		return nil, false, nil
	}
	return elements[index], true, nil
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
