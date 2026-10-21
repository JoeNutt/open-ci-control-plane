// SPDX-License-Identifier: GPL-3.0-or-later

package configuration_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/configuration"
)

func TestAConfigurationWithoutTheWebhookSecretIsRefusedNamingIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "controlplane.json")
	if err := os.WriteFile(path, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write the configuration: %v", err)
	}

	_, err := configuration.Read(path)

	if !errors.Is(err, configuration.ErrMissingItem) || !strings.Contains(err.Error(), "webhookSecretFile") {
		t.Fatalf("refusal is not a missing webhookSecretFile: %v", err)
	}
}
