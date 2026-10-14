// SPDX-License-Identifier: GPL-3.0-or-later

package domain

const checkRunNamePrefix = "policy/"

func CheckRunFor(decided FamilyDecision, id DecisionID) CheckRun {
	return CheckRun{Name: checkRunNamePrefix + string(decided.Family), Conclusion: ConclusionSuccess}
}
