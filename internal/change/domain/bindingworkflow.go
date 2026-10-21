// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

const MaxWorkflowFileBytes = 255

var ErrMissingWorkflow = errors.New("binding names no test workflow")

var ErrMalformedWorkflow = errors.New("workflow is not a workflow file name")

var workflowFileRule = segmentRule{
	maxBytes:        MaxWorkflowFileBytes,
	firstCharacters: asciiAlphanumerics,
	wholeCharacters: asciiAlphanumerics + "-._",
}

var workflowFileExtensions = []string{".yml", ".yaml"}

type WorkflowFile struct {
	value string
}

func ParseWorkflowFile(stated string) (WorkflowFile, error) {
	if stated == "" {
		return WorkflowFile{}, ErrMissingWorkflow
	}
	if !workflowFileRule.admits(stated) || !slices.ContainsFunc(workflowFileExtensions, func(extension string) bool {
		return strings.HasSuffix(stated, extension)
	}) {
		return WorkflowFile{}, fmt.Errorf("%w: %d bytes", ErrMalformedWorkflow, len(stated))
	}
	return WorkflowFile{value: stated}, nil
}

func (workflow WorkflowFile) String() string { return workflow.value }
