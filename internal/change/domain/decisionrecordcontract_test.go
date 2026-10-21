// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	httpContract         = "openapi.yaml"
	recordSchemaInScheme = "DecisionRecord"
)

type openAPIContract struct {
	Components struct {
		Schemas map[string]struct {
			Properties map[string]yaml.Node `yaml:"properties"`
		} `yaml:"schemas"`
	} `yaml:"components"`
}

func TestEveryFieldARecordPublishesIsOneTheHTTPContractDeclares(t *testing.T) {
	declared := declaredRecordProperties(t)
	decided := domain.FamilyDecision{
		Family:   domain.PolicyFamilyStructural,
		Decision: domain.UndecidedDecision(revisionForTest(t)),
	}
	id, err := domain.ParseDecisionID(wellFormedDecisionID)
	if err != nil {
		t.Fatalf("parse decision id: %v", err)
	}

	encoded, err := json.Marshal(domain.RecordFor(decided, id))
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var published map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &published); err != nil {
		t.Fatalf("decode: %v", err)
	}

	for field := range published {
		if _, isDeclared := declared[field]; !isDeclared {
			t.Errorf("the record publishes %q, which %s's %s does not declare",
				field, httpContract, recordSchemaInScheme)
		}
	}
}

func declaredRecordProperties(t *testing.T) map[string]yaml.Node {
	t.Helper()
	stated, err := os.ReadFile(filepath.Join(moduleRoot(t), httpContract))
	if err != nil {
		t.Fatalf("read %s: %v", httpContract, err)
	}
	var contract openAPIContract
	if err := yaml.Unmarshal(stated, &contract); err != nil {
		t.Fatalf("decode %s: %v", httpContract, err)
	}
	schema, isDeclared := contract.Components.Schemas[recordSchemaInScheme]
	if !isDeclared || len(schema.Properties) == 0 {
		t.Fatalf("%s declares no %s properties", httpContract, recordSchemaInScheme)
	}
	return schema.Properties
}
