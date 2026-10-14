// SPDX-License-Identifier: GPL-3.0-or-later

// Package domain holds the change document, the published contract policy is evaluated against.
package domain

import "encoding/json"

const SchemaID = "ci.diff/v1"

type Document struct {
	PR   PullRequest `json:"pr"`
	Diff Diff        `json:"diff"`
}

func (document Document) MarshalJSON() ([]byte, error) {
	type withoutMarshaller Document
	return json.Marshal(struct {
		SchemaVersion SchemaMajor `json:"schema_version"`
		withoutMarshaller
	}{CurrentSchemaMajor, withoutMarshaller(document)})
}

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

type Author struct {
	Login       string            `json:"login"`
	Association AuthorAssociation `json:"association"`
	IsBot       bool              `json:"is_bot"`
}

type Approval struct {
	Login    string `json:"login"`
	IsAuthor bool   `json:"is_author"`
}

type Diff struct {
	Paths                 Paths            `json:"paths"`
	Files                 Declared[File]   `json:"files"`
	Kinds                 Kinds            `json:"kinds"`
	Totals                Totals           `json:"totals"`
	ByDir                 PathsByDir       `json:"by_dir"`
	NewFiles              Paths            `json:"new_files"`
	Imports               Declared[Import] `json:"imports"`
	UnresolvedImportFiles int              `json:"unresolved_import_files"`
	ResolvedImportShare   float64          `json:"resolved_import_share"`
	ModuleGraphDelta      ModuleGraphDelta `json:"module_graph_delta"`
	Sensitive             Declared[string] `json:"sensitive"`
	Unknown               Declared[string] `json:"unknown"`
}

type Totals struct {
	Files      int `json:"files"`
	Additions  int `json:"additions"`
	Deletions  int `json:"deletions"`
	PatchBytes int `json:"patch_bytes"`
}

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

type File struct {
	Path       string     `json:"path"`
	Action     FileAction `json:"action"`
	Additions  int        `json:"additions"`
	Deletions  int        `json:"deletions"`
	PatchBytes int        `json:"patch_bytes"`
}

type Import struct {
	Path              string                `json:"path"`
	Language          *string               `json:"language"`
	Extractor         *string               `json:"extractor"`
	State             ImportState           `json:"state"`
	Refs              Declared[string]      `json:"refs"`
	DeclarationsAdded Declared[Declaration] `json:"declarations_added"`
}

type Declaration struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type ModuleGraphDelta struct {
	EdgesAdded   Declared[ModuleEdge] `json:"edges_added"`
	EdgesRemoved Declared[ModuleEdge] `json:"edges_removed"`
	Incomplete   bool                 `json:"incomplete"`
}

type ModuleEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}
