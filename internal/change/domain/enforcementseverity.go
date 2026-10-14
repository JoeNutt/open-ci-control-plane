// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "errors"

type EnforcementSeverity string

const (
	EnforcementSeverityDeny   EnforcementSeverity = "deny"
	EnforcementSeverityWarn   EnforcementSeverity = "warn"
	EnforcementSeverityDryRun EnforcementSeverity = "dry_run"
)

var declaredEnforcementSeverities = []EnforcementSeverity{
	EnforcementSeverityDeny,
	EnforcementSeverityWarn,
	EnforcementSeverityDryRun,
}

var ErrUnknownEnforcementSeverity = errors.New("unknown enforcement severity")

func ParseEnforcementSeverity(name string) (EnforcementSeverity, error) {
	return parseDeclaration(name, declaredEnforcementSeverities, ErrUnknownEnforcementSeverity)
}

func (severity *EnforcementSeverity) UnmarshalText(stated []byte) error {
	parsed, err := ParseEnforcementSeverity(string(stated))
	if err != nil {
		return err
	}
	*severity = parsed
	return nil
}
