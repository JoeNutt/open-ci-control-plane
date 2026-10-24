// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"slices"
	"testing"
)

func TestTestSetYieldsEachMemberInAscendingOrder(t *testing.T) {
	set := testSet{words: []uint64{0b101, 0, 1 << 63}}

	members := slices.Collect(set.members())

	if want := []int{0, 2, 191}; !slices.Equal(members, want) {
		t.Fatalf("members are %v, want %v", members, want)
	}
}

func TestTestSetSizeCountsItsMembers(t *testing.T) {
	set := testSet{words: []uint64{0b1011, 0, 1 << 63}}

	if size := set.size(); size != 4 {
		t.Fatalf("size is %d, want 4", size)
	}
}
