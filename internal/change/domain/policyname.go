// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "errors"

var ErrMalformedPolicyName = errors.New("policy name is not an identifier")

type PolicyName struct {
	value string
}

func ParsePolicyName(stated string) (PolicyName, error) {
	admitted, err := admittedPolicyIdentifier(stated, ErrMalformedPolicyName)
	if err != nil {
		return PolicyName{}, err
	}
	return PolicyName{value: admitted}, nil
}

func (name PolicyName) String() string { return name.value }
