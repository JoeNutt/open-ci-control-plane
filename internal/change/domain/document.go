// SPDX-License-Identifier: GPL-3.0-or-later

// Package domain holds the change document, the published contract that policy is
// evaluated against.
package domain

import (
	"encoding/json"
	"slices"
)

// SchemaMajor is a major version of the change document contract. Fields are added within
// a major version and never repurposed or renamed; a removal is a new major version.
type SchemaMajor int

// CurrentSchemaMajor is the change document major version this release publishes.
const CurrentSchemaMajor SchemaMajor = 1

// Document is the change document for one change, published to policy as ci.diff/v1.
type Document struct {
	PR   PullRequest `json:"pr"`
	Diff Diff        `json:"diff"`
}

// PullRequest is the context of a change: who proposed it, where, and against what.
type PullRequest struct {
	Number     int       `json:"number"`
	Repository string    `json:"repository"`
	BaseRef    string    `json:"base_ref"`
	HeadSHA    string    `json:"head_sha"`
	Draft      bool      `json:"draft"`
	Author     Author    `json:"author"`
	Labels     Labels    `json:"labels"`
	Approvals  Approvals `json:"approvals"`
}

type Author struct {
	Login       string `json:"login"`
	Association string `json:"association"`
	IsBot       bool   `json:"is_bot"`
}

type Labels []string

func (l Labels) MarshalJSON() ([]byte, error) {
	if len(l) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal([]string(l))
}

type Approval struct {
	Login    string `json:"login"`
	IsAuthor bool   `json:"is_author"`
}

type Approvals []Approval

func (a Approvals) MarshalJSON() ([]byte, error) {
	if len(a) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal([]Approval(a))
}

// MarshalJSON encodes the document, stamping it with the major version this release
// publishes rather than one the caller supplies.
func (d Document) MarshalJSON() ([]byte, error) {
	type fields Document
	return json.Marshal(struct {
		SchemaVersion SchemaMajor `json:"schema_version"`
		fields
	}{CurrentSchemaMajor, fields(d)})
}

// Diff is the artefact under decision: the paths a change touches and what it did to them.
type Diff struct {
	Paths Paths `json:"paths"`
}

// Paths is a list of changed file paths. It is never marshalled as null, because a field
// absent from the document evaluates to undefined in the policy engine and undefined
// deletes a control.
type Paths []string

// MarshalJSON encodes the paths as a JSON array, empty rather than null.
func (p Paths) MarshalJSON() ([]byte, error) {
	if len(p) == 0 {
		return []byte("[]"), nil
	}
	sorted := slices.Clone(p)
	slices.Sort(sorted)
	return json.Marshal([]string(sorted))
}
