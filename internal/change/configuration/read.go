// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

import (
	"errors"
	"fmt"
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
	listenAddressItem     item = "listenAddress"
	appIDItem             item = "appId"
	appKeyFileItem        item = "appKeyFile"
	webhookSecretFileItem item = "webhookSecretFile"
	installationIDsItem   item = "installationIds"
	familiesItem          item = "families"
	forgeBaseURLItem      item = "forgeBaseUrl"
)

var items = []item{
	listenAddressItem,
	appIDItem,
	appKeyFileItem,
	webhookSecretFileItem,
	installationIDsItem,
	familiesItem,
	forgeBaseURLItem,
}

var ErrMalformedConfiguration = errors.New("the configuration file is not a configuration")

type Configuration struct {
	listenAddress   netip.AddrPort
	appID           int64
	appKey          Secret
	webhookSecret   Secret
	installationIDs []int64
	families        []domain.PolicyFamily
	forgeBaseURL    domain.HTTPSOrigin
}

func Read(path string) (Configuration, error) {
	content, err := boundedContentOf(path, MaxFileBytes)
	if err != nil {
		return Configuration{}, fmt.Errorf("read the configuration file: %w", err)
	}
	stated, err := statedItemsIn(content)
	if err != nil {
		return Configuration{}, fmt.Errorf("%w: %w", ErrMalformedConfiguration, err)
	}
	return stated.configuration()
}

func (configured Configuration) ListenAddress() netip.AddrPort { return configured.listenAddress }

func (configured Configuration) AppID() int64 { return configured.appID }

func (configured Configuration) AppKey() Secret { return configured.appKey }

func (configured Configuration) WebhookSecret() Secret { return configured.webhookSecret }

func (configured Configuration) InstallationIDs() []int64 {
	return slices.Clone(configured.installationIDs)
}

func (configured Configuration) Families() []domain.PolicyFamily {
	return slices.Clone(configured.families)
}

func (configured Configuration) ForgeBaseURL() domain.HTTPSOrigin { return configured.forgeBaseURL }

func (stated statedItems) configuration() (Configuration, error) {
	address, addressErr := listenAddressIn(stated)
	appID, appIDErr := appIDIn(stated)
	appKey, appKeyErr := appKeyIn(stated)
	webhookSecret, webhookSecretErr := webhookSecretIn(stated)
	installationIDs, installationIDsErr := installationIDsIn(stated)
	families, familiesErr := familiesIn(stated)
	forgeBaseURL, forgeBaseURLErr := forgeBaseURLIn(stated)
	refusals := errors.Join(addressErr, appIDErr, appKeyErr, webhookSecretErr, installationIDsErr, familiesErr,
		forgeBaseURLErr)
	if refusals != nil {
		return Configuration{}, refusals
	}
	return Configuration{
		listenAddress:   address,
		appID:           appID,
		appKey:          appKey,
		webhookSecret:   webhookSecret,
		installationIDs: installationIDs,
		families:        families,
		forgeBaseURL:    forgeBaseURL,
	}, nil
}
