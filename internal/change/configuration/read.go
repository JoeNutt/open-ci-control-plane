// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"slices"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	MaxFileBytes       = 64 << 10
	MaxAppKeyFileBytes = 16 << 10
	MaxSecretFileBytes = 4 << 10
	MaxInstallations   = 256
)

const (
	listenAddress     item = "listenAddress"
	appID             item = "appId"
	appKeyFile        item = "appKeyFile"
	webhookSecretFile item = "webhookSecretFile"
	installationIDs   item = "installationIds"
	families          item = "families"
)

var ErrMalformedConfiguration = errors.New("the configuration file is not a configuration")

var errTrailingContent = errors.New("content follows the configuration object")

var items = []item{listenAddress, appID, appKeyFile, webhookSecretFile, installationIDs, families}

type Configuration struct {
	listenAddress   netip.AddrPort
	appID           int64
	appKey          Secret
	webhookSecret   Secret
	installationIDs []int64
	families        []domain.PolicyFamily
}

type statedConfiguration struct {
	ListenAddress     string   `json:"listenAddress"`
	AppID             *int64   `json:"appId"`
	AppKeyFile        string   `json:"appKeyFile"`
	WebhookSecretFile string   `json:"webhookSecretFile"`
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

func (configured Configuration) ListenAddress() netip.AddrPort { return configured.listenAddress }

func (configured Configuration) AppID() int64 { return configured.appID }

func (configured Configuration) AppKey() Secret { return configured.appKey }

func (configured Configuration) WebhookSecret() domain.WebhookSecret {
	return domain.WebhookSecret(configured.webhookSecret.Reveal())
}

func (configured Configuration) InstallationIDs() []int64 {
	return slices.Clone(configured.installationIDs)
}

func (configured Configuration) Families() []domain.PolicyFamily {
	return slices.Clone(configured.families)
}

func statedConfigurationIn(content []byte) (statedConfiguration, error) {
	if err := keysCheckedIn(content); err != nil {
		return statedConfiguration{}, malformed(err)
	}
	decoding := json.NewDecoder(bytes.NewReader(content))
	decoding.DisallowUnknownFields()
	var stated statedConfiguration
	if err := decoding.Decode(&stated); err != nil {
		return statedConfiguration{}, malformed(err)
	}
	if _, err := decoding.Token(); !errors.Is(err, io.EOF) {
		return statedConfiguration{}, malformed(errors.Join(errTrailingContent, err))
	}
	return stated, nil
}

func (stated statedConfiguration) configuration() (Configuration, error) {
	address, addressErr := listenAddressIn(stated.ListenAddress)
	app, appErr := appIDIn(stated.AppID)
	key, keyErr := appKeyFile.contentAt(stated.AppKeyFile, MaxAppKeyFileBytes)
	secret, secretErr := webhookSecretIn(stated.WebhookSecretFile)
	installations, installationsErr := installationIDsIn(stated.InstallationIDs)
	policyFamilies, familiesErr := familiesIn(stated.Families)
	if err := errors.Join(addressErr, appErr, keyErr, secretErr, installationsErr, familiesErr); err != nil {
		return Configuration{}, err
	}
	return Configuration{
		listenAddress:   address,
		appID:           app,
		appKey:          secretOf(key),
		webhookSecret:   secret,
		installationIDs: installations,
		families:        policyFamilies,
	}, nil
}

func malformed(reason error) error {
	return fmt.Errorf("%w: %w", ErrMalformedConfiguration, reason)
}
