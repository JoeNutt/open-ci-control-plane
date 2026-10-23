// SPDX-License-Identifier: GPL-3.0-or-later

// Package rules reads path classification rules from outside the governed repository.
package rules

import (
	"errors"
	"fmt"
	"io"

	"go.yaml.in/yaml/v3"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var ErrMalformedRuleDocument = errors.New("classification rule document is malformed")

func ReadFrom(stated io.Reader) (domain.ClassificationRules, error) {
	reading := yaml.NewDecoder(stated)
	reading.KnownFields(true)

	var written map[domain.PathKind][]string
	if err := reading.Decode(&written); err != nil {
		return domain.ClassificationRules{}, fmt.Errorf("%w: %w", ErrMalformedRuleDocument, err)
	}
	return domain.ParseClassificationRules(written)
}

