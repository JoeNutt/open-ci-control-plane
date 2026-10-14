// SPDX-License-Identifier: GPL-3.0-or-later

package github

import "fmt"

func ParseChangeSource(settings Settings) (ChangeSource, error) {
	forge, err := parseConnection(settings)
	if err != nil {
		return ChangeSource{}, err
	}
	if settings.Client.Timeout <= 0 {
		return ChangeSource{}, fmt.Errorf("%w: %v", ErrUnboundedClient, settings.Client.Timeout)
	}
	return ChangeSource{forge: forge}, nil
}
