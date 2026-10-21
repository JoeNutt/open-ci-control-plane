// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	MaxFileBytes       = 64 << 10
	MaxSecretFileBytes = 4 << 10
	MaxAppKeyFileBytes = 16 << 10
)

const (
	webhookSecretFile item = "webhookSecretFile"
	appKeyFile        item = "appKeyFile"
	installationIDs   item = "installationIds"
)

const (
	textFileEnd         = "\n"
	firstPrintableASCII = ' '
	lastPrintableASCII  = '~'
)

var ErrMalformedConfiguration = errors.New("the configuration file is not a configuration")

var ErrOversizedFile = errors.New("the file exceeds its byte bound")

var errEmptyFile = errors.New("the file is empty")

var errEmptySecret = errors.New("the secret is empty once its line break is removed")

var errUnprintableSecret = errors.New("the secret holds a character outside printable ASCII")

type Configuration struct {
	webhookSecret   Secret
	appKey          Secret
	installationIDs []int64
}

type statedConfiguration struct {
	WebhookSecretFile string  `json:"webhookSecretFile"`
	AppKeyFile        string  `json:"appKeyFile"`
	InstallationIDs   []int64 `json:"installationIds"`
}

func Read(path string) (Configuration, error) {
	content, err := boundedContentOf(path, MaxFileBytes)
	if err != nil {
		return Configuration{}, fmt.Errorf("read the configuration file: %w", err)
	}
	stated, err := statedConfigurationIn(content)
	if err != nil {
		return Configuration{}, err
	}
	return stated.configuration()
}

func (configured Configuration) WebhookSecret() domain.WebhookSecret {
	return domain.WebhookSecret(configured.webhookSecret.Reveal())
}

func (configured Configuration) AppKey() Secret { return configured.appKey }

func (configured Configuration) InstallationIDs() []int64 {
	return slices.Clone(configured.installationIDs)
}

func statedConfigurationIn(content []byte) (statedConfiguration, error) {
	var stated statedConfiguration
	if err := json.Unmarshal(content, &stated); err != nil {
		return statedConfiguration{}, fmt.Errorf("%w: %w", ErrMalformedConfiguration, err)
	}
	return stated, nil
}

func (stated statedConfiguration) configuration() (Configuration, error) {
	secret, secretErr := webhookSecretIn(stated.WebhookSecretFile)
	key, keyErr := appKeyFile.contentAt(stated.AppKeyFile, MaxAppKeyFileBytes)
	installations, installationsErr := installationIDsIn(stated.InstallationIDs)
	if err := errors.Join(secretErr, keyErr, installationsErr); err != nil {
		return Configuration{}, err
	}
	return Configuration{
		webhookSecret:   secret,
		appKey:          secretOf(key),
		installationIDs: installations,
	}, nil
}

func webhookSecretIn(path string) (Secret, error) {
	content, err := webhookSecretFile.contentAt(path, MaxSecretFileBytes)
	if err != nil {
		return Secret{}, err
	}
	secret := bytes.TrimSuffix(content, []byte(textFileEnd))
	if len(secret) == 0 {
		return Secret{}, webhookSecretFile.invalid(errEmptySecret)
	}
	if bytes.ContainsFunc(secret, isControlOrBeyondASCII) {
		return Secret{}, webhookSecretFile.invalid(errUnprintableSecret)
	}
	return secretOf(secret), nil
}

func installationIDsIn(stated []int64) ([]int64, error) {
	if len(stated) == 0 {
		return nil, installationIDs.missing()
	}
	return slices.Clone(stated), nil
}

func isControlOrBeyondASCII(character rune) bool {
	return character < firstPrintableASCII || character > lastPrintableASCII
}

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
