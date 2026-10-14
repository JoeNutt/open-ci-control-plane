// SPDX-License-Identifier: GPL-3.0-or-later

package domain

func CheckRunFor(decided FamilyDecision, id DecisionID) CheckRun {
	return CheckRun{Name: "policy/" + string(decided.Family), Conclusion: ConclusionSuccess}
}
