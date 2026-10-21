// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type GroupStatement struct {
	Name  string
	Order int
}

type PolicyGroup struct {
	name  GroupName
	order int
}

func (group PolicyGroup) Name() GroupName { return group.name }

func (group PolicyGroup) Order() int { return group.order }
