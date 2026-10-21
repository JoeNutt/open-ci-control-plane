package ci.structural_test

import data.ci.structural

policy := {
	"catalog": {"clean-architecture": {
		"rule": "direction",
		"direction": "lower_order_may_not_import_higher",
		"groups": [
			{"name": "domain", "order": 0},
			{"name": "application", "order": 1},
			{"name": "adapter", "order": 2},
		],
	}},
	"bindings": {"JoeNutt/open-ci-control-plane": {"policies": [{
		"use": "clean-architecture",
		"import_roots": [{"prefix": module_root, "path": ""}],
		"paths": {
			"domain": ["internal/change/domain/**"],
			"application": ["internal/change/application/**"],
			"adapter": ["internal/change/forge/**"],
		},
	}]}},
}

module_root := "github.com/JoeNutt/open-ci-control-plane"

domain_imports_forge := {"from": "internal/change/domain/gate.go", "to": $"{module_root}/internal/change/forge/github"}

change_adding(edges) := {
	"schema_version": 1,
	"pr": {"repository": "JoeNutt/open-ci-control-plane"},
	"diff": {"module_graph_delta": {"edges_added": edges, "edges_removed": [], "incomplete": false}},
}

test_an_added_edge_into_a_higher_order_group_is_refused_naming_file_import_and_groups if {
	decided := structural.decision with input as change_adding([domain_imports_forge]) with data.ci.policy as policy

	decided.effect == "block"
	[violation] := decided.violations
	violation.message == concat("", [
		"internal/change/domain/gate.go imports ",
		"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github: ",
		`group "domain" may not import group "adapter"`,
	])
}

test_a_refusal_is_stated_in_the_decision_contract_the_engine_parses if {
	decided := structural.decision with input as change_adding([domain_imports_forge]) with data.ci.policy as policy

	decided.input_wellformed == true
	decided.schema_version_accepted == 1
	decided.reviewers == []
	decided.fail_closed == true
	[violation] := decided.violations
	violation.rule_id == "clean-architecture"
	violation.severity == "deny"
}

test_a_policy_whose_direction_the_rule_does_not_know_is_undecided_never_admitted if {
	unknown_direction := json.patch(policy, [{
		"op": "replace",
		"path": "/catalog/clean-architecture/direction",
		"value": "higher_order_may_not_import_lower",
	}])

	not structural.decision with input as change_adding([domain_imports_forge]) with data.ci.policy as unknown_direction
}
