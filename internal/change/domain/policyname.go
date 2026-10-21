// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type PolicyName struct {
	value string
}

func ParsePolicyName(stated string) (PolicyName, error) {
	return PolicyName{value: stated}, nil
}

func (name PolicyName) String() string { return name.value }
