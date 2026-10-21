// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "encoding/json"

type MaskedInput struct {
	tree map[string]any
}

func MaskedInputOf(document Document) (MaskedInput, error) {
	encoded, err := json.Marshal(document)
	if err != nil {
		return MaskedInput{}, err
	}
	var tree map[string]any
	if err := json.Unmarshal(encoded, &tree); err != nil {
		return MaskedInput{}, err
	}
	for _, stated := range document.Diff.Sensitive {
		pointer, err := ParseJSONPointer(stated)
		if err != nil {
			return MaskedInput{}, err
		}
		tokens := pointer.Tokens()
		parent := tree
		for _, token := range tokens[:len(tokens)-1] {
			parent = parent[token].(map[string]any)
		}
		delete(parent, tokens[len(tokens)-1])
	}
	return MaskedInput{tree: tree}, nil
}

func (masked MaskedInput) MarshalJSON() ([]byte, error) { return json.Marshal(masked.tree) }
