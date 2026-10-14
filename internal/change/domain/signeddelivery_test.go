// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	deliveredBody = `{"action":"opened","number":412}`
	appSecret     = "s3cret"
	trueSignature = "sha256=51e90d780f2a25ff6328cb24733ed9c4b05b70202696d272ab38890b3f016416"
)

func TestParseSignedDeliveryCarriesABodySignedWithTheAppSecret(t *testing.T) {
	delivered, err := domain.ParseSignedDelivery(
		[]byte(deliveredBody), trueSignature, appSecret)

	if err != nil {
		t.Fatalf("parse a correctly signed delivery: %v", err)
	}
	if string(delivered.Body()) != deliveredBody {
		t.Errorf("body is %q, want %q", delivered.Body(), deliveredBody)
	}
}
