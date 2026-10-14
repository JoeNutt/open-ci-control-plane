// SPDX-License-Identifier: GPL-3.0-or-later

package github

func ParseChangeSource(settings Settings) (ChangeSource, error) {
	forge, err := parseConnection(settings)
	if err != nil {
		return ChangeSource{}, err
	}
	return ChangeSource{forge: forge}, nil
}
