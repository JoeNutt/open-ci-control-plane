// SPDX-License-Identifier: GPL-3.0-or-later

package domain

func isLowerHex(digits string) bool {
	for index := 0; index < len(digits); index++ {
		if !isLowerHexDigit(digits[index]) {
			return false
		}
	}
	return true
}

func isLowerHexDigit(character byte) bool {
	return ('0' <= character && character <= '9') || ('a' <= character && character <= 'f')
}
