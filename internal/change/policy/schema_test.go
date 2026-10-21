// SPDX-License-Identifier: GPL-3.0-or-later

package policy_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"testing"
)

const (
	schemaDirectory   = "../../../schema/"
	catalogSchemaPath = schemaDirectory + "policy.catalog.v1.schema.json"
)

func TestTheLiveCatalogMatchesThePublishedCatalogSchema(t *testing.T) {
	if faults := schemaFaultsOf(t, string(fixture(t, liveCatalogPath)), catalogSchemaPath); len(faults) != 0 {
		t.Errorf("the live catalog does not match %s: %v", catalogSchemaPath, faults)
	}
}

func TestThePublishedCatalogSchemaRefusesEveryCatalogRefusedByItsShape(t *testing.T) {
	for form, refusal := range catalogsRefusedByTheirShape(t) {
		t.Run(form, func(t *testing.T) {
			if faults := schemaFaultsOf(t, refusal.written, catalogSchemaPath); len(faults) == 0 {
				t.Errorf("a catalog with %s matches %s", form, catalogSchemaPath)
			}
		})
	}
}

func schemaFaultsOf(t *testing.T, written, schemaPath string) []string {
	t.Helper()

	matching := exec.Command("opa", "eval", "-I", "--format=raw",
		`json.match_schema(yaml.unmarshal(input.written), input.schema)`)
	matching.Stdin = bytes.NewReader(schemaQueryFor(t, written, schemaPath))
	answered, err := matching.CombinedOutput()
	if err != nil {
		t.Fatalf("opa eval against %s: %v\n%s", schemaPath, err, answered)
	}
	return faultsIn(t, answered)
}

func schemaQueryFor(t *testing.T, written, schemaPath string) []byte {
	t.Helper()

	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read %s: %v", schemaPath, err)
	}
	query, err := json.Marshal(struct {
		Written string          `json:"written"`
		Schema  json.RawMessage `json:"schema"`
	}{Written: written, Schema: schema})
	if err != nil {
		t.Fatalf("marshal the file and %s: %v", schemaPath, err)
	}
	return query
}

func faultsIn(t *testing.T, answered []byte) []string {
	t.Helper()

	var pair []json.RawMessage
	if err := json.Unmarshal(answered, &pair); err != nil || len(pair) != 2 {
		t.Fatalf("opa answered %s, want a match and its faults", answered)
	}
	var matched bool
	var faults []struct {
		Error string `json:"error"`
	}
	if err := errors.Join(json.Unmarshal(pair[0], &matched), json.Unmarshal(pair[1], &faults)); err != nil {
		t.Fatalf("opa answered %s: %v", answered, err)
	}
	if matched != (len(faults) == 0) {
		t.Fatalf("opa answered a match of %t with %d faults: %s", matched, len(faults), answered)
	}
	described := make([]string, 0, len(faults))
	for _, fault := range faults {
		described = append(described, fault.Error)
	}
	return described
}
