// SPDX-License-Identifier: GPL-3.0-or-later

package policy

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/engine/opa"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/engine/opa/structural"
)

const (
	bundleManifestFile = ".manifest"
	policyDataFile     = "ci/policy/data.json"
	bundleFileMode     = 0o600
)

var errUnshippedWord = errors.New("the model holds a rule or direction with no word to serve it by")

type BundleHandler struct {
	Active *ActivePolicy
}

func (handler BundleHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.URL.Path != opa.BundleResourceFor(domain.PolicyFamilyStructural) {
		http.NotFound(writer, request)
		return
	}
	archive, err := handler.servedBundle()
	if err != nil {
		handler.Active.log.Error(unservedBundleMessage, reasonLogKey, err.Error())
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if _, err := writer.Write(archive); err != nil {
		handler.Active.log.Error(unservedBundleMessage, reasonLogKey, err.Error())
	}
}

const noPolicyRevision = "0000000000000000000000000000000000000000"

func (handler BundleHandler) servedBundle() ([]byte, error) {
	accepted, err := handler.Active.Accepted()
	if errors.Is(err, ErrNoPolicyLoaded) {
		return noPolicyBundle()
	}
	if err != nil {
		return nil, err
	}
	return bundleOf(accepted)
}

func noPolicyBundle() ([]byte, error) {
	files, err := ruleFilesAt(noPolicyRevision)
	if err != nil {
		return nil, err
	}
	return gzippedTarOf(files)
}

func ruleFilesAt(revision string) (map[string][]byte, error) {
	manifest, err := json.Marshal(bundleManifest{Revision: revision})
	if err != nil {
		return nil, fmt.Errorf("encode the bundle manifest: %w", err)
	}
	files := map[string][]byte{bundleManifestFile: manifest}
	for name, module := range structural.Modules() {
		files[name] = []byte(module)
	}
	return files, nil
}

const unservedBundleMessage = "policy bundle not served"

func bundleOf(accepted AcceptedRevision) ([]byte, error) {
	files, err := bundleFilesOf(accepted)
	if err != nil {
		return nil, err
	}
	return gzippedTarOf(files)
}

func bundleFilesOf(accepted AcceptedRevision) (map[string][]byte, error) {
	data, err := policyDataOf(accepted.Model())
	if err != nil {
		return nil, err
	}
	encodedData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("encode the policy data: %w", err)
	}
	files, err := ruleFilesAt(accepted.Revision().String())
	if err != nil {
		return nil, err
	}
	files[policyDataFile] = encodedData
	return files, nil
}

type bundleManifest struct {
	Revision string `json:"revision"`
}

func gzippedTarOf(files map[string][]byte) ([]byte, error) {
	var archive bytes.Buffer
	zipped := gzip.NewWriter(&archive)
	tarred := tar.NewWriter(zipped)
	for _, name := range slices.Sorted(maps.Keys(files)) {
		if err := writeTarFile(tarred, name, files[name]); err != nil {
			return nil, err
		}
	}
	if err := errors.Join(tarred.Close(), zipped.Close()); err != nil {
		return nil, fmt.Errorf("close the bundle archive: %w", err)
	}
	return archive.Bytes(), nil
}

func writeTarFile(tarred *tar.Writer, name string, content []byte) error {
	if err := tarred.WriteHeader(&tar.Header{Name: name, Mode: bundleFileMode, Size: int64(len(content))}); err != nil {
		return fmt.Errorf("write the bundle header of %s: %w", name, err)
	}
	if _, err := tarred.Write(content); err != nil {
		return fmt.Errorf("write the bundle file %s: %w", name, err)
	}
	return nil
}
