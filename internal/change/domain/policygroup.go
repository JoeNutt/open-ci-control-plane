// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "fmt"

type GroupStatement struct {
	Name  string
	Order int
}

type PolicyGroup struct {
	name  GroupName
	order int
}

func parsePolicyGroup(stated GroupStatement) (PolicyGroup, error) {
	name, err := ParseGroupName(stated.Name)
	if err != nil {
		return PolicyGroup{}, fmt.Errorf("group: %w", err)
	}
	return PolicyGroup{name: name, order: stated.Order}, nil
}

func (group PolicyGroup) Name() GroupName { return group.name }

func (group PolicyGroup) Order() int { return group.order }
