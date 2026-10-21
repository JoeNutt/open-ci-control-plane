// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const MaxFileBytes = 64 << 10

const webhookSecretFileItem = "webhookSecretFile"

var ErrMissingItem = errors.New("the configuration is missing a required item")

var ErrMalformedConfiguration = errors.New("the configuration file is not a configuration")

var ErrOversizedFile = errors.New("the file exceeds its byte bound")

type Configuration struct {
	webhookSecret domain.WebhookSecret
}

type statedConfiguration struct {
	WebhookSecretFile string `json:"webhookSecretFile"`
}

func Read(path string) (Configuration, error) {
	content, err := boundedContentOf(path, MaxFileBytes)
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
	secret, err := os.ReadFile(stated.WebhookSecretFile)
	if err != nil {
		return Configuration{}, err
	}
	return Configuration{webhookSecret: domain.WebhookSecret(secret)}, nil
}

func (read Configuration) WebhookSecret() domain.WebhookSecret { return read.webhookSecret }

func boundedContentOf(path string, maxBytes int) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	content, readErr := io.ReadAll(io.LimitReader(file, int64(maxBytes)+1))
	if err := errors.Join(readErr, file.Close()); err != nil {
		return nil, err
	}
	if len(content) > maxBytes {
		return nil, fmt.Errorf("%w: more than %d bytes", ErrOversizedFile, maxBytes)
	}
	return content, nil
}
