// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseWorkflowFileKeepsTheFileName(t *testing.T) {
	for _, stated := range []string{"open-ci-tests.yml", "open-ci-tests.yaml"} {
		parsed, err := domain.ParseWorkflowFile(stated)
		if err != nil {
			t.Fatalf("parse workflow %q: %v", stated, err)
		}
		if parsed.String() != stated {
			t.Errorf("workflow is %q, want %q", parsed, stated)
		}
	}
}

func TestParseWorkflowFileRefusesAMissingWorkflow(t *testing.T) {
	if _, err := domain.ParseWorkflowFile(""); !errors.Is(err, domain.ErrMissingWorkflow) {
		t.Fatalf("an empty workflow gave %v, want ErrMissingWorkflow", err)
	}
}

func TestParseWorkflowFileRefusesWhatIsNotAWorkflowFileName(t *testing.T) {
	refused := map[string]string{
		"a directory":         ".github/workflows/open-ci-tests.yml",
		"not YAML":            "open-ci-tests.json",
		"only an extension":   ".yml",
		"a space":             "open ci.yml",
		"one byte over bound": strings.Repeat("a", domain.MaxWorkflowFileBytes) + ".yml",
	}
	for form, stated := range refused {
		t.Run(form, func(t *testing.T) {
			if _, err := domain.ParseWorkflowFile(stated); !errors.Is(err, domain.ErrMalformedWorkflow) {
				t.Fatalf("%s gave %v, want ErrMalformedWorkflow", form, err)
			}
		})
	}
}
