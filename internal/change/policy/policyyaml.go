// SPDX-License-Identifier: GPL-3.0-or-later

package policy

import (
	"bytes"
	"errors"
	"io"
	"slices"

	"go.yaml.in/yaml/v3"
)

var ErrMalformedPolicyFile = errors.New("policy file is not YAML of the declared shape")

var ErrAliasOrAnchor = errors.New("policy file contains a YAML alias or anchor")

var ErrSecondDocument = errors.New("policy file holds more than one YAML document")

type policyDocument interface {
	catalogDocument
}

func decodedStrictly[Document policyDocument](content []byte) (Document, error) {
	var refused Document
	if err := refuseAliasOrAnchor(content); err != nil {
		return refused, err
	}
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(true)
	var document Document
	if err := decoder.Decode(&document); err != nil {
		return refused, ErrMalformedPolicyFile
	}
	var second yaml.Node
	if err := decoder.Decode(&second); !errors.Is(err, io.EOF) {
		return refused, ErrSecondDocument
	}
	return document, nil
}

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
