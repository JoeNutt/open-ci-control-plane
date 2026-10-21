// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	MaxFileBytes       = 64 << 10
	MaxSecretFileBytes = 4 << 10
	MaxAppKeyFileBytes = 16 << 10
	MaxInstallations   = 256
)

const (
	webhookSecretFile item = "webhookSecretFile"
	appKeyFile        item = "appKeyFile"
	installationIDs   item = "installationIds"
	families          item = "families"
)

var ErrMalformedConfiguration = errors.New("the configuration file is not a configuration")

type Configuration struct {
	webhookSecret   Secret
	appKey          Secret
	installationIDs []int64
	families        []domain.PolicyFamily
}

type statedConfiguration struct {
	WebhookSecretFile string   `json:"webhookSecretFile"`
	AppKeyFile        string   `json:"appKeyFile"`
	InstallationIDs   []int64  `json:"installationIds"`
	Families          []string `json:"families"`
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

func (configured Configuration) Families() []domain.PolicyFamily { return configured.families }

func castFamilies(stated []string) []domain.PolicyFamily {
	var out []domain.PolicyFamily
	for _, name := range stated {
		out = append(out, domain.PolicyFamily(name))
	}
	return out
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
	if err := errors.Join(secretErr, keyErr, installationsErr, familiesPresentIn(stated.Families)); err != nil {
		return Configuration{}, err
	}
	return Configuration{
		webhookSecret:   secret,
		appKey:          secretOf(key),
		installationIDs: installations,
		families:        castFamilies(stated.Families),
	}, nil
}
