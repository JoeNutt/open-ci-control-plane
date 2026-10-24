// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type basePath struct {
	path     RepositoryPath
	isAtBase bool
}

func atBaseUnder(path RepositoryPath) basePath { return basePath{path: path, isAtBase: true} }

func absentAtBase() basePath { return basePath{} }
