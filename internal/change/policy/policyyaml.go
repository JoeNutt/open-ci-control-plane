// SPDX-License-Identifier: GPL-3.0-or-later

package policy

import (
	"errors"
	"slices"

	"go.yaml.in/yaml/v3"
)

var ErrMalformedPolicyFile = errors.New("policy file is not YAML of the declared shape")

var ErrAliasOrAnchor = errors.New("policy file contains a YAML alias or anchor")

func refuseAliasOrAnchor(content []byte) error {
	var root yaml.Node
	if err := yaml.Unmarshal(content, &root); err != nil {
		return ErrMalformedPolicyFile
	}
	if containsAliasOrAnchor(&root) {
		return ErrAliasOrAnchor
	}
	return nil
}

func containsAliasOrAnchor(node *yaml.Node) bool {
	if node.Kind == yaml.AliasNode || node.Anchor != "" {
		return true
	}
	return slices.ContainsFunc(node.Content, containsAliasOrAnchor)
}
