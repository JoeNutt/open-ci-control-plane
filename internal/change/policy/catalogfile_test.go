// SPDX-License-Identifier: GPL-3.0-or-later

package policy_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/policy"
)

func TestParseCatalogFileReadsTheLiveCatalog(t *testing.T) {
	parsed, err := policy.ParseCatalogFile(fixture(t, liveCatalogPath))
	if err != nil {
		t.Fatalf("parse catalog file: %v", err)
	}
	if parsed.Name().String() != "clean-architecture" {
		t.Errorf("name is %q, want clean-architecture", parsed.Name())
	}
	if parsed.Direction() != domain.LowerOrderMayNotImportHigher {
		t.Errorf("direction is %v, want lower order may not import higher", parsed.Direction())
	}
	groups := parsed.Groups()
	if len(groups) != 4 || groups[3].Name().String() != "driver" || groups[3].Order() != 3 {
		t.Errorf("groups are %v, want four ending in driver at 3", groups)
	}
}

func TestParseCatalogFileRefusesAFileItCannotTrust(t *testing.T) {
	refused := map[string]struct {
		written string
		reason  error
	}{
		"not yaml at all": {written: "name: [clean\n", reason: policy.ErrMalformedPolicyFile},
		"an unknown rule word": {
			written: liveCatalogWith(t, "rule: direction", "rule: layering"),
			reason:  domain.ErrUnknownPolicyRule,
		},
		"an unknown direction word": {
			written: liveCatalogWith(t, "lower_order_may_not_import_higher", "higher_may_import_lower"),
			reason:  domain.ErrUnknownPolicyDirection,
		},
		"an unknown key": {
			written: liveCatalogWith(t, "rule: direction", "rule: direction\nowner: platform"),
			reason:  policy.ErrMalformedPolicyFile,
		},
		"a YAML anchor": {
			written: liveCatalogWith(t, "name: clean-architecture", "name: &policy clean-architecture"),
			reason:  policy.ErrAliasOrAnchor,
		},
		"a YAML alias": {
			written: liveCatalogWith(t, "rule: direction", "rule: &rule direction\nx: *rule"),
			reason:  policy.ErrAliasOrAnchor,
		},
		"a second YAML document": {
			written: string(fixture(t, liveCatalogPath)) + "---\nname: shadow\n",
			reason:  policy.ErrSecondDocument,
		},
		"a later schema version": {
			written: liveCatalogWith(t, "schema_version: 1", "schema_version: 2"),
			reason:  policy.ErrUnsupportedSchemaVersion,
		},
		"no schema version": {
			written: liveCatalogWith(t, "schema_version: 1\n", ""),
			reason:  policy.ErrUnsupportedSchemaVersion,
		},
		"a group with no order": {
			written: liveCatalogWith(t, "{ name: domain, order: 0 }", "{ name: domain }"),
			reason:  policy.ErrMissingKey,
		},
	}
	for form, refusal := range refused {
		t.Run(form, func(t *testing.T) {
			if _, err := policy.ParseCatalogFile([]byte(refusal.written)); !errors.Is(err, refusal.reason) {
				t.Fatalf("%s gave %v, want %v", form, err, refusal.reason)
			}
		})
	}
}

func liveCatalogWith(t *testing.T, written, replacement string) string {
	t.Helper()

	return fixtureWith(t, liveCatalogPath, written, replacement)
}
