// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseGroupNameKeepsTheName(t *testing.T) {
	name, err := domain.ParseGroupName("domain")
	if err != nil {
		t.Fatalf("parse group name: %v", err)
	}
	if name.String() != "domain" {
		t.Errorf("name is %q, want domain", name)
	}
}

func TestParseGroupNameRefusesWhatIsNotAnIdentifier(t *testing.T) {
	for form, stated := range malformedIdentifiers() {
		t.Run(form, func(t *testing.T) {
			if _, err := domain.ParseGroupName(stated); !errors.Is(err, domain.ErrMalformedGroupName) {
				t.Fatalf("%s gave %v, want ErrMalformedGroupName", form, err)
			}
		})
	}
}
