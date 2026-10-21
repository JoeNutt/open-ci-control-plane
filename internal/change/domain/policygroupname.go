// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
)

var ErrMalformedGroupName = errors.New("group name is not an identifier")

type GroupName struct {
	value string
}

func ParseGroupName(stated string) (GroupName, error) {
	if !policyIdentifierRule.admits(stated) {
		return GroupName{}, fmt.Errorf("%w: %d bytes", ErrMalformedGroupName, len(stated))
	}
	return GroupName{value: stated}, nil
}

func (name GroupName) String() string { return name.value }
