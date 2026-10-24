// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"reflect"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestNewTestedFileHoldsEachHitLineOnceInAscendingOrder(t *testing.T) {
	path := mustParseRepositoryPath(t, heldFile)

	tested := domain.NewTestedFile(path, []uint32{9, 3, 9})

	if want := domain.NewTestedFile(path, []uint32{3, 9}); !reflect.DeepEqual(tested, want) {
		t.Fatalf("tested file is %+v, want %+v", tested, want)
	}
}
