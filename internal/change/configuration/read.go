// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

const MaxFileBytes = 64 << 10

const webhookSecretFileItem = "webhookSecretFile"

var ErrMissingItem = errors.New("the configuration is missing a required item")

var ErrMalformedConfiguration = errors.New("the configuration file is not a configuration")

type Configuration struct{}

type statedConfiguration struct {
	WebhookSecretFile string `json:"webhookSecretFile"`
}

func Read(path string) (Configuration, error) {
	file, err := os.Open(path)
	if err != nil {
		return Configuration{}, fmt.Errorf("read the configuration file: %w", err)
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, MaxFileBytes+1))
	if err != nil {
		return Configuration{}, fmt.Errorf("read the configuration file: %w", err)
	}
	if len(content) > MaxFileBytes {
		return Configuration{}, errors.New("too big")
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
