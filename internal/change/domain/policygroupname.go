// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "errors"

var ErrMalformedGroupName = errors.New("group name is not an identifier")

type GroupName struct {
	value string
}

func ParseGroupName(stated string) (GroupName, error) {
	admitted, err := admittedPolicyIdentifier(stated, ErrMalformedGroupName)
	if err != nil {
		return GroupName{}, err
	}
	return GroupName{value: admitted}, nil
}

func (name GroupName) String() string { return name.value }
