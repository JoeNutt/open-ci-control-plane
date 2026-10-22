// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type CheckRun struct {
	Name           string
	Conclusion     Conclusion
	Title          string
	Headline       string
	DecisionID     DecisionID
	Revision       BundleRevision
	Violations     []CheckRunViolation
	ViolationCount int
}

type CheckRunViolation struct {
	RuleID     string
	Message    string
	OwningTeam string
	DocsURL    string
	Marking    string
}
