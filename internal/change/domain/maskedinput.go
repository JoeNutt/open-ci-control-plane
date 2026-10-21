// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "encoding/json"

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
	parent := tree
	for _, token := range tokens[:len(tokens)-1] {
		child, ok := parent[token].(map[string]any)
		if !ok {
			return
		}
		parent = child
	}
	delete(parent, tokens[len(tokens)-1])
}
