// SPDX-License-Identifier: GPL-3.0-or-later

package policy

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/engine/opa"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/engine/opa/structural"
)

type BundleHandler struct {
	Active *ActivePolicy
}

func (handler BundleHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	accepted, err := handler.Active.Accepted()
	if err != nil || request.URL.Path != opa.BundleResourceFor(domain.PolicyFamilyStructural) {
		http.NotFound(writer, request)
		return
	}
	files := structural.Modules()
	manifest, _ := json.Marshal(map[string]string{"revision": accepted.Revision().String()})
	files[".manifest"] = string(manifest)
	data, _ := json.Marshal(policyDataOf(accepted.Model()))
	files["ci/policy/data.json"] = string(data)
	var archive bytes.Buffer
	zipped := gzip.NewWriter(&archive)
	tarred := tar.NewWriter(zipped)
	for name, content := range files {
		_ = tarred.WriteHeader(&tar.Header{Name: "/" + name, Mode: 0o600, Size: int64(len(content))})
		_, _ = tarred.Write([]byte(content))
	}
	_ = tarred.Close()
	_ = zipped.Close()
	_, _ = writer.Write(archive.Bytes())
}

type policyData struct {
	Catalog  map[string]catalogData `json:"catalog"`
	Bindings map[string]bindingData `json:"bindings"`
}

type catalogData struct {
	Rule      string      `json:"rule"`
	Direction string      `json:"direction"`
	Groups    []groupData `json:"groups"`
}

type groupData struct {
	Name  string `json:"name"`
	Order int    `json:"order"`
}

type bindingData struct {
	Policies []boundPolicyData `json:"policies"`
}

type boundPolicyData struct {
	Use         string              `json:"use"`
	ImportRoots []importRootData    `json:"import_roots"`
	Paths       map[string][]string `json:"paths"`
}

type importRootData struct {
	Prefix string `json:"prefix"`
	Path   string `json:"path"`
}

func policyDataOf(model domain.PolicyModel) policyData {
	data := policyData{Catalog: map[string]catalogData{}, Bindings: map[string]bindingData{}}
	for _, catalogued := range model.Catalog() {
		var groups []groupData
		for _, group := range catalogued.Groups() {
			groups = append(groups, groupData{Name: group.Name().String(), Order: group.Order()})
		}
		data.Catalog[catalogued.Name().String()] = catalogData{
			Rule: "direction", Direction: "lower_order_may_not_import_higher", Groups: groups,
		}
	}
	for _, binding := range model.Bindings() {
		var policies []boundPolicyData
		for _, bound := range binding.Policies() {
			var roots []importRootData
			for _, root := range bound.ImportRoots() {
				roots = append(roots, importRootData{Prefix: root.Prefix().String(), Path: root.Directory().String()})
			}
			paths := map[string][]string{}
			for _, group := range bound.Groups() {
				for _, pattern := range group.Patterns() {
					paths[group.Group().String()] = append(paths[group.Group().String()], pattern.Glob())
				}
			}
			policies = append(policies, boundPolicyData{Use: bound.Use().String(), ImportRoots: roots, Paths: paths})
		}
		data.Bindings[binding.Repository().String()] = bindingData{Policies: policies}
	}
	return data
}
