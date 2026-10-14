// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type PathKind string

const (
	PathKindTest         PathKind = "test"
	PathKindAuthz        PathKind = "authz"
	PathKindRoute        PathKind = "route"
	PathKindIaC          PathKind = "iac"
	PathKindCIConfig     PathKind = "ci_config"
	PathKindPolicyConfig PathKind = "policy_config"
	PathKindLockfile     PathKind = "lockfile"
	PathKindMigration    PathKind = "migration"
	PathKindGenerated    PathKind = "generated"
)

var declaredPathKinds = []PathKind{
	PathKindTest,
	PathKindAuthz,
	PathKindRoute,
	PathKindIaC,
	PathKindCIConfig,
	PathKindPolicyConfig,
	PathKindLockfile,
	PathKindMigration,
	PathKindGenerated,
}
