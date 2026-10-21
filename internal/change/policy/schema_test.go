// SPDX-License-Identifier: GPL-3.0-or-later

package policy_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"testing"
)

const catalogSchemaPath = "../../../schema/policy.catalog.v1.schema.json"

func TestTheLiveCatalogMatchesThePublishedCatalogSchema(t *testing.T) {
	if faults := schemaFaultsOf(t, string(fixture(t, liveCatalogPath)), catalogSchemaPath); len(faults) != 0 {
		t.Errorf("the live catalog does not match %s: %v", catalogSchemaPath, faults)
	}
}

func schemaFaultsOf(t *testing.T, written, schemaPath string) []string {
	t.Helper()

	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read %s: %v", schemaPath, err)
	}
	asked, err := json.Marshal(struct {
		Written string          `json:"written"`
		Schema  json.RawMessage `json:"schema"`
	}{Written: written, Schema: schema})
	if err != nil {
		t.Fatalf("marshal the file and %s: %v", schemaPath, err)
	}
	matching := exec.Command("opa", "eval", "-I", "--format=raw",
		`json.match_schema(yaml.unmarshal(input.written), input.schema)`)
	matching.Stdin = bytes.NewReader(asked)
	answered, err := matching.CombinedOutput()
	if err != nil {
		t.Fatalf("opa eval against %s: %v\n%s", schemaPath, err, answered)
	}
	var reported []json.RawMessage
	if err := json.Unmarshal(answered, &reported); err != nil || len(reported) != 2 {
		t.Fatalf("opa answered %s", answered)
	}
	var faults []struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(reported[1], &faults); err != nil {
		t.Fatalf("decode the faults: %v", err)
	}
	described := make([]string, 0, len(faults))
	for _, fault := range faults {
		described = append(described, fault.Error)
	}
	return described
}
