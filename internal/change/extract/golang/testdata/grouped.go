package fixture

import (
	"context"
	_ "embed"

	gate "github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var _ = context.Background
var _ gate.Document
