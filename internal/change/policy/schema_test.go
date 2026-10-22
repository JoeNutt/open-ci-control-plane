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
	schemaDirectory      = "../../../schema/"
	catalogSchemaPath    = schemaDirectory + "policy.catalog.v1.schema.json"
	bindingSchemaPath    = schemaDirectory + "policy.binding.v1.schema.json"
	exampleBindingPath   = fixtureDirectory + "section13/binding.yaml"
	section13CatalogPath = fixtureDirectory + "section13/catalog.yaml"
)

func TestEachPublishedSchemaAdmitsTheLiveAndSection13Files(t *testing.T) {
	admittedBySchema := map[string][]string{
		catalogSchemaPath: {liveCatalogPath},
		bindingSchemaPath: {liveBindingPath, exampleBindingPath},
	}
	for schemaPath, admitted := range admittedBySchema {
		for _, path := range admitted {
			if faults := schemaFaultsOf(t, string(fixture(t, path)), schemaPath); len(faults) != 0 {
				t.Errorf("%s does not match %s: %v", path, schemaPath, faults)
			}
		}
	}
}

func TestEachPublishedSchemaRefusesEveryFileRefusedByItsShape(t *testing.T) {
	refusedBySchema := map[string]map[string]refusedFile{
		catalogSchemaPath: catalogsRefusedByTheirShape(t),
		bindingSchemaPath: bindingsRefusedByTheirShape(t),
	}
	for schemaPath, refused := range refusedBySchema {
		t.Run(schemaPath, func(t *testing.T) { expectEachRefusedBy(t, schemaPath, refused) })
	}
}

func expectEachRefusedBy(t *testing.T, schemaPath string, refused map[string]refusedFile) {
	t.Helper()

	for form, refusal := range refused {
		t.Run(form, func(t *testing.T) {
			if faults := schemaFaultsOf(t, refusal.written, schemaPath); len(faults) == 0 {
				t.Errorf("a file with %s matches %s", form, schemaPath)
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
