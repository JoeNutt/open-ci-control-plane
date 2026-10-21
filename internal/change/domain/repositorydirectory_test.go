// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseRepositoryDirectoryKeepsTheDirectory(t *testing.T) {
	for _, stated := range []string{"", "internal/change"} {
		parsed, err := domain.ParseRepositoryDirectory(stated)
		if err != nil {
			t.Fatalf("parse directory %q: %v", stated, err)
		}
		if parsed.String() != stated {
			t.Errorf("directory is %q, want %q", parsed, stated)
		}
	}
}

func TestParseRepositoryDirectoryRefusesWhatIsNotRepositoryRelative(t *testing.T) {
	refused := map[string]string{
		"absolute":            "/internal/change",
		"a trailing slash":    "internal/change/",
		"an empty segment":    "internal//change",
		"a parent segment":    "internal/../..",
		"a current segment":   "./internal",
		"one byte over bound": strings.Repeat("a", domain.DeclaredLimits().MaxPathBytes+1),
	}
	for form, stated := range refused {
		t.Run(form, func(t *testing.T) {
			if _, err := domain.ParseRepositoryDirectory(stated); !errors.Is(err, domain.ErrMalformedRepositoryDirectory) {
				t.Fatalf("%s gave %v, want ErrMalformedRepositoryDirectory", form, err)
			}
		})
	}
}
