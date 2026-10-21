// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

const webhookSecretFileItem = "webhookSecretFile"

var ErrMissingItem = errors.New("the configuration is missing a required item")

var ErrMalformedConfiguration = errors.New("the configuration file is not a configuration")

type Configuration struct{}

type statedConfiguration struct {
	WebhookSecretFile string `json:"webhookSecretFile"`
}

func Read(path string) (Configuration, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Configuration{}, fmt.Errorf("read the configuration file: %w", err)
	}
	var stated statedConfiguration
	if err := json.Unmarshal(content, &stated); err != nil {
		return Configuration{}, fmt.Errorf("%w: %w", ErrMalformedConfiguration, err)
	}
	if stated.WebhookSecretFile == "" {
		return Configuration{}, fmt.Errorf("%w: %s", ErrMissingItem, webhookSecretFileItem)
	}
	return Configuration{}, nil
}
