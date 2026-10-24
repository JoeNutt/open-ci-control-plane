// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"iter"
	"math/bits"
)

const testsPerWord = 64

type testSet struct {
	words []uint64
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
