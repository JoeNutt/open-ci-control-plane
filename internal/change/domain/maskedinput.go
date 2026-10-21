// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"encoding/json"
	"strconv"
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
		removeAt(tree, pointer)
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

func removeAt(tree map[string]any, pointer JSONPointer) {
	tokens := pointer.Tokens()
	var node any = tree
	for _, token := range tokens[:len(tokens)-1] {
		child, exists := childOf(node, token)
		if !exists {
			return
		}
		node = child
	}
	parent, isObject := node.(map[string]any)
	if !isObject {
		return
	}
	delete(parent, tokens[len(tokens)-1])
}

func childOf(node any, token string) (any, bool) {
	switch container := node.(type) {
	case map[string]any:
		child, exists := container[token]
		return child, exists
	case []any:
		index, err := strconv.Atoi(token)
		if err != nil || index < 0 || index >= len(container) {
			return nil, false
		}
		return container[index], true
	default:
		return nil, false
	}
}
