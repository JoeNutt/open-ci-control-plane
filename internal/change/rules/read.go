// SPDX-License-Identifier: GPL-3.0-or-later

// Package rules reads path classification rules from outside the governed repository.
package rules

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var ErrMalformedRuleDocument = errors.New("classification rule document is malformed")

type document struct {
	Kinds map[domain.PathKind][]string `json:"kinds"`
}

func ReadFrom(stated io.Reader) (domain.ClassificationRules, error) {
	reading := json.NewDecoder(stated)
	reading.DisallowUnknownFields()

	var read document
	if err := reading.Decode(&read); err != nil {
		return domain.ClassificationRules{}, fmt.Errorf("%w: %w", ErrMalformedRuleDocument, err)
	}
	return domain.ParseClassificationRules(read.Kinds)
}
