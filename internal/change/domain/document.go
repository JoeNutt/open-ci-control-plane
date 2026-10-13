// SPDX-License-Identifier: GPL-3.0-or-later

// Package domain holds the change document, the published contract that policy is
// evaluated against.
package domain

import "encoding/json"

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

// MarshalJSON encodes the document, stamping it with the major version this release
// publishes rather than one the caller supplies.
func (d Document) MarshalJSON() ([]byte, error) {
	type fields Document
	return json.Marshal(struct {
		SchemaVersion SchemaMajor `json:"schema_version"`
		fields
	}{CurrentSchemaMajor, fields(d)})
}

// PullRequest is the context of a change: who proposed it, where, and against what.
type PullRequest struct {
	Number     int                `json:"number"`
	Repository string             `json:"repository"`
	BaseRef    string             `json:"base_ref"`
	HeadSHA    string             `json:"head_sha"`
	Draft      bool               `json:"draft"`
	Author     Author             `json:"author"`
	Labels     Declared[string]   `json:"labels"`
	Approvals  Declared[Approval] `json:"approvals"`
}

// Author is the account that proposed a change.
type Author struct {
	Login       string `json:"login"`
	Association string `json:"association"`
	IsBot       bool   `json:"is_bot"`
}

// Approval is one review approval recorded against a change.
type Approval struct {
	Login    string `json:"login"`
	IsAuthor bool   `json:"is_author"`
}

// Diff is the artefact under decision: the paths a change touches and what it did to them.
type Diff struct {
	Paths Paths          `json:"paths"`
	Files Declared[File] `json:"files"`
	Kinds Kinds          `json:"kinds"`
}

// Kinds holds the changed paths of one kind per bucket. Every bucket is published on every
// evaluation, so a policy reading one never sees undefined.
type Kinds struct {
	Test         Paths `json:"test"`
	Authz        Paths `json:"authz"`
	Route        Paths `json:"route"`
	IaC          Paths `json:"iac"`
	CIConfig     Paths `json:"ci_config"`
	PolicyConfig Paths `json:"policy_config"`
	Lockfile     Paths `json:"lockfile"`
	Migration    Paths `json:"migration"`
	Generated    Paths `json:"generated"`
}

// File is one changed file: what the change did to it, and how much of it moved.
type File struct {
	Path       string     `json:"path"`
	Action     FileAction `json:"action"`
	Additions  int        `json:"additions"`
	Deletions  int        `json:"deletions"`
	PatchBytes int        `json:"patch_bytes"`
}
