// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"slices"
	"testing"
)

var threeWordMembers = []int{0, 1, 3, 2*testsPerWord + testsPerWord - 1}

func TestTestSetYieldsEachMemberInAscendingOrder(t *testing.T) {
	members := membersOf(threeWordSet())

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
	revision := &indexRevision{testCount: testsPerWord + 1}
	one := testSet{revision: revision, words: []uint64{0b0011, 0}}
	other := testSet{revision: revision, words: []uint64{0b0110, 1}}

	joined, err := one.union(other)

	if want := []int{0, 1, 2, testsPerWord}; err != nil || !slices.Equal(membersOf(joined), want) {
		t.Fatalf("union is %v (%v), want %v", membersOf(joined), err, want)
	}
	if !slices.Equal(membersOf(one), []int{0, 1}) {
		t.Fatalf("the union changed the set it was taken from")
	}
}

func TestTestSetUnionIsRefusedAcrossTwoIndexRevisions(t *testing.T) {
	one := testSet{revision: &indexRevision{testCount: 1}, words: []uint64{1}}
	other := testSet{revision: &indexRevision{testCount: 1}, words: []uint64{1}}

	joined, err := one.union(other)

	if !errors.Is(err, errMixedRevisions) {
		t.Fatalf("joined %v (%v), want errMixedRevisions", membersOf(joined), err)
	}
}

func TestTestSetHoldsMembersEitherSideOfAWordBoundary(t *testing.T) {
	for _, asked := range []struct {
		testCount int
		members   []uint32
		words     int
	}{
		{testCount: 0, members: nil, words: 0},
		{testCount: testsPerWord, members: []uint32{testsPerWord - 1}, words: 1},
		{testCount: testsPerWord + 1, members: []uint32{testsPerWord - 1, testsPerWord}, words: 2},
	} {
		set := testSetOf(&indexRevision{testCount: asked.testCount}, asked.members)

		want := positionsOf(asked.members)
		if len(set.words) != asked.words || set.size() != len(want) || !slices.Equal(membersOf(set), want) {
			t.Errorf("%d tests holding %v: %d words, size %d, members %v", asked.testCount, asked.members, len(set.words), set.size(), membersOf(set))
		}
	}
}

func positionsOf(members []uint32) []int {
	var positions []int
	for _, member := range members {
		positions = append(positions, int(member))
	}
	return positions
}

func membersOf(set testSet) []int { return slices.Collect(set.members()) }

func threeWordSet() testSet {
	return testSet{words: []uint64{0b1011, 0, 1 << (testsPerWord - 1)}}
}
