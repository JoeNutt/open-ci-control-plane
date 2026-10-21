// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

import "errors"

type Configuration struct{}

func Read(path string) (Configuration, error) {
	return Configuration{}, errors.New("missing webhookSecretFile")
}
