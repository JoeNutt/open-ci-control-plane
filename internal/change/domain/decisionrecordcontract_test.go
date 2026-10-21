// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"go.yaml.in/yaml/v3"
)

const (
	httpContract         = "openapi.yaml"
	decisionRecordSchema = "DecisionRecord"
)

type openAPIContract struct {
	Components struct {
		Schemas map[string]struct {
			Properties map[string]struct{} `yaml:"properties"`
		} `yaml:"schemas"`
	} `yaml:"components"`
}

func TestEveryFieldARecordPublishesIsOneTheHTTPContractDeclares(t *testing.T) {
	declared := declaredRecordProperties(t)

	var published map[string]json.RawMessage
	if err := json.Unmarshal(encodedRecordOf(t, undecidedStructuralDecision(t)), &published); err != nil {
		t.Fatalf("decode: %v", err)
	}

	for field := range published {
		if _, isDeclared := declared[field]; !isDeclared {
			t.Errorf("the record publishes %q, which %s's %s does not declare",
				field, httpContract, decisionRecordSchema)
		}
	}
}

func declaredRecordProperties(t *testing.T) map[string]struct{} {
	t.Helper()
	stated, err := os.ReadFile(filepath.Join(moduleRoot(t), httpContract))
	if err != nil {
		t.Fatalf("read %s: %v", httpContract, err)
	}
	var contract openAPIContract
	if err := yaml.Unmarshal(stated, &contract); err != nil {
		t.Fatalf("decode %s: %v", httpContract, err)
	}
	schema, isDeclared := contract.Components.Schemas[decisionRecordSchema]
	if !isDeclared || len(schema.Properties) == 0 {
		t.Fatalf("%s declares no %s properties", httpContract, decisionRecordSchema)
	}
	return schema.Properties
}
