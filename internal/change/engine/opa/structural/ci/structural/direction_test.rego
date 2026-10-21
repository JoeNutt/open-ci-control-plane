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
	"bindings": {control_plane: {"policies": [{
		"use": "clean-architecture",
		"import_roots": [{"prefix": module_root, "path": ""}],
		"paths": {
			"domain": ["internal/change/domain/**"],
			"application": ["internal/change/application/**"],
			"adapter": ["internal/change/forge/**"],
		},
	}]}},
}

control_plane := "JoeNutt/open-ci-control-plane"

module_root := $"github.com/{control_plane}"

control_plane_binding := $"/bindings/{replace(control_plane, "/", "~1")}"

control_plane_policies := $"{control_plane_binding}/policies"

domain_imports_forge := {"from": "internal/change/domain/gate.go", "to": $"{module_root}/internal/change/forge/github"}

change_adding(edges) := change_in(control_plane, edges)

change_in(repository, edges) := {
	"schema_version": 1,
	"pr": {"repository": repository},
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

policy_also_rooting(root) := json.patch(policy, [{
	"op": "add",
	"path": $"{control_plane_policies}/0/import_roots/-",
	"value": root,
}])

test_the_longest_matching_import_root_places_the_import if {
	forge_vendored_elsewhere := policy_also_rooting({
		"prefix": $"{module_root}/internal/change/forge",
		"path": "third_party/forge",
	})

	decided := structural.decision with input as change_adding([domain_imports_forge])
		with data.ci.policy as forge_vendored_elsewhere

	decided.effect == "allow"
	decided.violations == []
}

test_an_import_root_matches_whole_path_segments_only if {
	vendored_into_forge := policy_also_rooting({
		"prefix": "example.com/vendored",
		"path": "internal/change/forge/vendored",
	})
	domain_imports_lookalike := {"from": domain_imports_forge.from, "to": "example.com/vendoredextra/thing"}

	decided := structural.decision with input as change_adding([domain_imports_lookalike])
		with data.ci.policy as vendored_into_forge

	decided.effect == "allow"
	decided.violations == []
}

test_an_import_under_no_declared_root_is_outside_every_group if {
	domain_imports_unrooted := {"from": domain_imports_forge.from, "to": "internal/change/forge/github"}

	decided := structural.decision with input as change_adding([domain_imports_unrooted]) with data.ci.policy as policy

	decided.effect == "allow"
	decided.violations == []
}

test_a_package_in_no_group_at_either_end_is_never_refused[note] if {
	some note, edge in {
		"importer in no group": {"from": "cmd/tool/main.go", "to": domain_imports_forge.to},
		"imported in no group": {"from": domain_imports_forge.from, "to": $"{module_root}/internal/change/unplaced"},
	}

	decided := structural.decision with input as change_adding([edge]) with data.ci.policy as policy

	decided.effect == "allow"
	decided.violations == []
}

mobile_app := "acme/mobile-app"

model_imports_view := {"from": "app/model/user.go", "to": "example.com/mobile/app/view"}

policy_with_a_second_binding := object.union(policy, {
	"catalog": {"mvvm": {
		"rule": "direction",
		"direction": "lower_order_may_not_import_higher",
		"groups": [
			{"name": "model", "order": 0},
			{"name": "viewmodel", "order": 1},
			{"name": "view", "order": 2},
		],
	}},
	"bindings": {mobile_app: {"policies": [{
		"use": "mvvm",
		"import_roots": [{"prefix": "example.com/mobile", "path": ""}],
		"paths": {
			"model": ["app/model/**"],
			"viewmodel": ["app/viewmodel/**"],
			"view": ["app/view/**"],
		},
	}]}},
})

test_each_repository_is_judged_only_by_the_policies_its_own_binding_names[note] if {
	some note, case in {
		"clean-architecture edge in the mobile app": {
			"repository": mobile_app,
			"edge": domain_imports_forge,
			"refusals": set(),
		},
		"mvvm edge in the mobile app": {
			"repository": mobile_app,
			"edge": model_imports_view,
			"refusals": {[
				"mvvm",
				`app/model/user.go imports example.com/mobile/app/view: group "model" may not import group "view"`,
			]},
		},
		"mvvm edge in the control plane": {
			"repository": control_plane,
			"edge": model_imports_view,
			"refusals": set(),
		},
	}

	decided := structural.decision with input as change_in(case.repository, [case.edge])
		with data.ci.policy as policy_with_a_second_binding

	{[violation.rule_id, violation.message] | some violation in decided.violations} == case.refusals
}

test_policy_data_the_rule_cannot_judge_by_leaves_the_decision_undefined[note] if {
	some note, patch in {
		"no binding for the repository": [{"op": "remove", "path": control_plane_binding}],
		"a binding naming a policy absent from the catalog": [{"op": "remove", "path": "/catalog/clean-architecture"}],
		"a binding naming no policies": [{"op": "replace", "path": control_plane_policies, "value": []}],
		"a direction the rule does not know": [{
			"op": "replace",
			"path": "/catalog/clean-architecture/direction",
			"value": "higher_order_may_not_import_lower",
		}],
		"no policy data at all": [{"op": "replace", "path": "", "value": {}}],
	}

	unjudgeable := json.patch(policy, patch)

	not structural.decision with input as change_adding([domain_imports_forge]) with data.ci.policy as unjudgeable
}
