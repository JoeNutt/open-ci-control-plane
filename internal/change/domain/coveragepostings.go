// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"cmp"
	"slices"
	"strings"
)

type coverageEdge struct {
	path RepositoryPath
	line uint32
	test uint32
}

type itemSpan struct {
	first int
	end   int
}

type coveragePostings struct {
	itemsOfFile map[RepositoryPath]itemSpan
	lineOfItem  []uint32
	offsets     []uint32
	tests       []uint32
}

func edgesOf(numbered []suiteTest) []coverageEdge {
	var edges []coverageEdge
	for dense, each := range numbered {
		edges = each.test.appendEdges(edges, uint32(dense))
	}
	return edges
}

func (test ReportedTest) appendEdges(edges []coverageEdge, dense uint32) []coverageEdge {
	for _, file := range test.files {
		for _, line := range file.hitLines {
			edges = append(edges, coverageEdge{path: file.path, line: line, test: dense})
		}
	}
	return edges
}

func postingsOf(edges []coverageEdge) coveragePostings {
	slices.SortFunc(edges, itemThenTestOrder)
	postings := coveragePostings{itemsOfFile: map[RepositoryPath]itemSpan{}, tests: make([]uint32, 0, len(edges))}
	for position, edge := range edges {
		if position == 0 || !edges[position-1].isOnItemOf(edge) {
			postings.openItem(edge.path, uint32(position))
			postings.lineOfItem = append(postings.lineOfItem, edge.line)
		}
		postings.tests = append(postings.tests, edge.test)
	}
	postings.offsets = append(postings.offsets, uint32(len(edges)))
	return postings
}

func itemThenTestOrder(one, other coverageEdge) int {
	return cmp.Or(
		strings.Compare(one.path.String(), other.path.String()),
		cmp.Compare(one.line, other.line),
		cmp.Compare(one.test, other.test),
	)
}

func (edge coverageEdge) isOnItemOf(other coverageEdge) bool {
	return edge.path == other.path && edge.line == other.line
}

func (postings *coveragePostings) openItem(path RepositoryPath, firstPosting uint32) {
	item := len(postings.offsets)
	postings.offsets = append(postings.offsets, firstPosting)
	span, isHeld := postings.itemsOfFile[path]
	if !isHeld {
		span.first = item
	}
	span.end = item + 1
	postings.itemsOfFile[path] = span
}

func (postings coveragePostings) testsOnLines(span itemSpan, lines []uint32) []uint32 {
	var tests []uint32
	for _, line := range lines {
		if item, isCovered := postings.itemOnLine(span, line); isCovered {
			tests = append(tests, postings.testsOn(item)...)
		}
	}
	return tests
}

func (postings coveragePostings) itemOnLine(span itemSpan, line uint32) (itemSpan, bool) {
	offset, isCovered := slices.BinarySearch(postings.lineOfItem[span.first:span.end], line)
	item := span.first + offset
	return itemSpan{first: item, end: item + 1}, isCovered
}

func (postings coveragePostings) testsOn(span itemSpan) []uint32 {
	return postings.tests[postings.offsets[span.first]:postings.offsets[span.end]]
}
