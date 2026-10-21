package ci.structural

decision := {
	"input_wellformed": input_wellformed,
	"schema_version_accepted": 1,
	"effect": effect,
	"violations": sort([violation | some violation in violations]),
	"reviewers": [],
	"fail_closed": true,
} if {
	every bound in bound_policies {
		refused_relation_by_direction[data.ci.policy.catalog[bound.use].direction]
	}
}

bound_policies := data.ci.policy.bindings[input.pr.repository].policies

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
	some bound in bound_policies
	model := data.ci.policy.catalog[bound.use]
	some edge in input.diff.module_graph_delta.edges_added
	importer := group_containing(bound, edge.from)
	imported := group_containing(bound, go_file_in(package_path(bound, edge.to)))
	refused_relation_by_direction[model.direction] == relation_of(order_of(model, importer), order_of(model, imported))
}

refused_relation_by_direction := {"lower_order_may_not_import_higher": "lower"}

relation_of(importer_order, imported_order) := "lower" if importer_order < imported_order

relation_of(importer_order, imported_order) := "same" if importer_order == imported_order

relation_of(importer_order, imported_order) := "higher" if importer_order > imported_order

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
	root := root_of(bound, imported)
}

root_of(bound, imported) := root if {
	roots := [candidate | some candidate in bound.import_roots; startswith(imported, candidate.prefix)]
	some root in roots
	count(root.prefix) == max({count(candidate.prefix) | some candidate in roots})
}

go_file_in(directory) := $"{directory}/package.go"

order_of(model, name) := group.order if {
	some group in model.groups
	group.name == name
}
