package ci.structural

decision := {
	"input_wellformed": input_wellformed,
	"schema_version_accepted": 1,
	"effect": effect,
	"violations": sort([violation | some violation in violations]),
	"reviewers": [],
	"fail_closed": true,
}

default input_wellformed := false

input_wellformed if {
	is_string(input.pr.repository)
	is_array(input.diff.module_graph_delta.edges_added)
}

effect := "block" if count(violations) > 0

effect := "allow" if count(violations) == 0

violations contains {
	"rule_id": bound.use,
	"message": refusal_of(edge, importer, imported),
	"severity": "deny",
} if {
	some bound in data.ci.policy.bindings[input.pr.repository].policies
	model := data.ci.policy.catalog[bound.use]
	some edge in input.diff.module_graph_delta.edges_added
	importer := group_containing(bound, edge.from)
	imported := group_containing(bound, go_file_in(package_path(bound, edge.to)))
	forbids(model.direction, order_of(model, importer), order_of(model, imported))
}

refusal_of(edge, importer, imported) := concat(": ", [
	$"{edge.from} imports {edge.to}",
	$`group "{importer}" may not import group "{imported}"`,
])

group_containing(bound, file) := group if {
	some group, patterns in bound.paths
	some pattern in patterns
	glob.match(pattern, ["/"], file)
}

package_path(bound, imported) := trim_prefix($"{root.path}{trim_prefix(imported, root.prefix)}", "/") if {
	some root in bound.import_roots
	startswith(imported, root.prefix)
}

go_file_in(directory) := $"{directory}/package.go"

order_of(model, name) := group.order if {
	some group in model.groups
	group.name == name
}

forbids("lower_order_may_not_import_higher", importer_order, imported_order) if importer_order < imported_order
