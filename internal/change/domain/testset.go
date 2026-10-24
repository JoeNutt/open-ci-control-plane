// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"iter"
	"math/bits"
	"slices"
)

const testsPerWord = 64

var errMixedRevisions = errors.New("the test sets come from two revisions of the coverage index")

type indexRevision struct {
	testCount int
}

type testSet struct {
	revision *indexRevision
	words    []uint64
}

func testSetOf(revision *indexRevision, members []uint32) testSet {
	words := make([]uint64, wordsHolding(revision.testCount))
	for _, member := range members {
		words[member/testsPerWord] |= 1 << (member % testsPerWord)
	}
	return testSet{revision: revision, words: words}
}

func wordsHolding(testCount int) int {
	return (testCount + testsPerWord - 1) / testsPerWord
}

func (set testSet) union(other testSet) (testSet, error) {
	if set.revision != other.revision {
		return testSet{}, errMixedRevisions
	}
	joined := slices.Clone(set.words)
	for position, word := range other.words {
		joined[position] |= word
	}
	return testSet{revision: set.revision, words: joined}, nil
}

func (set testSet) size() int {
	count := 0
	for _, word := range set.words {
		count += bits.OnesCount64(word)
	}
	return count
}

func (set testSet) members() iter.Seq[int] {
	return func(yield func(int) bool) {
		for position, word := range set.words {
			for ; word != 0; word &= word - 1 {
				if !yield(position*testsPerWord + bits.TrailingZeros64(word)) {
					return
				}
			}
		}
	}
}
