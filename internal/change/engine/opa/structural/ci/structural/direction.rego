package ci.structural

decision := {
	"effect": "block",
	"violations": [{"message": `internal/change/domain/gate.go imports github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github: group "domain" may not import group "adapter"`}],
}
