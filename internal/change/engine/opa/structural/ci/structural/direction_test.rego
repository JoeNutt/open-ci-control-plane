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

test_an_added_edge_the_direction_allows_passes if {
	forge_imports_domain := {
		"from": "internal/change/forge/github/client.go",
		"to": $"{module_root}/internal/change/domain",
	}

	decided := structural.decision with input as change_adding([forge_imports_domain]) with data.ci.policy as policy

	decided.effect == "allow"
	decided.violations == []
}

test_a_forbidden_import_the_edited_file_already_had_is_not_judged if {
	gate_still_imports_forge := object.union(change_adding([]), {"diff": {"imports": [{
		"path": domain_imports_forge.from,
		"language": "go",
		"extractor": "go/parser",
		"state": "resolved",
		"refs": [domain_imports_forge.to],
		"declarations_added": [],
	}]}})

	decided := structural.decision with input as gate_still_imports_forge with data.ci.policy as policy

	decided.effect == "allow"
	decided.violations == []
}

test_the_longest_matching_import_root_places_the_import if {
	forge_vendored_elsewhere := json.patch(policy, [{
		"op": "add",
		"path": "/bindings/JoeNutt~1open-ci-control-plane/policies/0/import_roots/-",
		"value": {"prefix": $"{module_root}/internal/change/forge", "path": "third_party/forge"},
	}])

	decided := structural.decision with input as change_adding([domain_imports_forge])
		with data.ci.policy as forge_vendored_elsewhere

	decided.effect == "allow"
	decided.violations == []
}

test_a_policy_whose_direction_the_rule_does_not_know_is_undecided_never_admitted if {
	unknown_direction := json.patch(policy, [{
		"op": "replace",
		"path": "/catalog/clean-architecture/direction",
		"value": "higher_order_may_not_import_lower",
	}])

	not structural.decision with input as change_adding([domain_imports_forge]) with data.ci.policy as unknown_direction
}
