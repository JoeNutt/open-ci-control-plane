// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"encoding/json"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const maskingSentinel = "sentinel-value"

type publishedMaskableFields struct {
	PR struct {
		Author struct {
			Login string `json:"login"`
		} `json:"author"`
		Approvals []struct {
			Login string `json:"login"`
		} `json:"approvals"`
	} `json:"pr"`
	Analyzers struct {
		Findings []struct {
			Path string `json:"path"`
		} `json:"findings"`
	} `json:"analyzers"`
}

func TestEachMaskablePointerClearsThePublishedFieldItNames(t *testing.T) {
	published := []struct {
		pointer string
		valueIn func(fields publishedMaskableFields) string
	}{
		{"/pr/author/login", func(fields publishedMaskableFields) string { return fields.PR.Author.Login }},
		{"/pr/approvals/0/login", func(fields publishedMaskableFields) string {
			if len(fields.PR.Approvals) == 0 {
				return ""
			}
			return fields.PR.Approvals[0].Login
		}},
		{"/analyzers/findings/0", func(fields publishedMaskableFields) string {
			if len(fields.Analyzers.Findings) == 0 {
				return ""
			}
			return fields.Analyzers.Findings[0].Path
		}},
	}

	for _, field := range published {
		t.Run(field.pointer, func(t *testing.T) {
			document := documentHoldingTheSentinelEverywhereMaskable()
			if field.valueIn(publishedFieldsOf(t, document)) != maskingSentinel {
				t.Fatalf("the encoded document does not publish the sentinel at %s", field.pointer)
			}

			document.Diff.Sensitive = domain.Declared[string]{field.pointer}
			masked, err := domain.MaskedInputOf(document)
			if err != nil {
				t.Fatalf("mask: %v", err)
			}
			if field.valueIn(publishedFieldsOf(t, masked)) == maskingSentinel {
				t.Errorf("masking %s left the sentinel published there", field.pointer)
			}
		})
	}
}

func documentHoldingTheSentinelEverywhereMaskable() domain.Document {
	document := domain.Document{}
	document.PR.Author.Login = maskingSentinel
	document.PR.Approvals = domain.Declared[domain.Approval]{{Login: maskingSentinel}}
	document.Analyzers.Findings = domain.Declared[domain.Finding]{{Path: maskingSentinel}}
	return document
}

func publishedFieldsOf(t *testing.T, encodable json.Marshaler) publishedMaskableFields {
	t.Helper()
	encoded, err := encodable.MarshalJSON()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var fields publishedMaskableFields
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return fields
}
