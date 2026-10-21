// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

type Secret struct {
	content *string
}

func secretOf(content []byte) Secret {
	held := string(content)
	return Secret{content: &held}
}

func (held Secret) Reveal() []byte { return []byte(*held.content) }
