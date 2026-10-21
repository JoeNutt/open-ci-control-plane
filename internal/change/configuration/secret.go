// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

import "fmt"

type Secret struct {
	content *string
}

func (Secret) Format(state fmt.State, _ rune) {
	fmt.Fprint(state, "[redacted]")
}

func (held Secret) Reveal() []byte { return []byte(*held.content) }
