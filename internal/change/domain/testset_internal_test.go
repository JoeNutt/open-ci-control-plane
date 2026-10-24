// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"slices"
	"testing"
)

var threeWordMembers = []int{0, 1, 3, 2*testsPerWord + testsPerWord - 1}

func TestTestSetYieldsEachMemberInAscendingOrder(t *testing.T) {
	members := slices.Collect(threeWordSet().members())

	if !slices.Equal(members, threeWordMembers) {
		t.Fatalf("members are %v, want %v", members, threeWordMembers)
	}
}

func TestTestSetSizeCountsItsMembers(t *testing.T) {
	if size := threeWordSet().size(); size != len(threeWordMembers) {
		t.Fatalf("size is %d, want %d", size, len(threeWordMembers))
	}
}

func TestTestSetUnionHoldsTheMembersOfBoth(t *testing.T) {
	one := testSet{words: []uint64{0b0011, 0}}
	other := testSet{words: []uint64{0b0110, 1}}

	joined, err := one.union(other)

	if want := []int{0, 1, 2, testsPerWord}; err != nil || !slices.Equal(slices.Collect(joined.members()), want) {
		t.Fatalf("union is %v (%v), want %v", slices.Collect(joined.members()), err, want)
	}
	if !slices.Equal(slices.Collect(one.members()), []int{0, 1}) {
		t.Fatalf("the union changed the set it was taken from")
	}
}

func threeWordSet() testSet {
	return testSet{words: []uint64{0b1011, 0, 1 << (testsPerWord - 1)}}
}
