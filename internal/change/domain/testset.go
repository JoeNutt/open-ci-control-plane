// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"iter"
	"math/bits"
)

type testSet struct {
	words []uint64
}

func (set testSet) members() iter.Seq[int] {
	return func(yield func(int) bool) {
		for i, word := range set.words {
			for ; word != 0; word &= word - 1 {
				if !yield(i*64 + bits.TrailingZeros64(word)) {
					return
				}
			}
		}
	}
}
