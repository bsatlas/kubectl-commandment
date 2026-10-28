/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"path"
	"path/filepath"
	"strings"
)

type copyFileSpec struct {
	PodName      string
	PodNamespace string
	File         copyPathSpec
}

type copyPathSpec interface {
	String() string
}

type copyLocalPath struct {
	file string
}

func newCopyLocalPath(fileName string) copyLocalPath {
	file := stripCopyTrailingSlash(fileName)
	path := copyLocalPath{
		file: file,
	}
	return path
}

func (p copyLocalPath) String() string { return p.file }

func (p copyLocalPath) Dir() copyLocalPath { return newCopyLocalPath(filepath.Dir(p.file)) }

func (p copyLocalPath) Base() copyLocalPath { return newCopyLocalPath(filepath.Base(p.file)) }

func (p copyLocalPath) Clean() copyLocalPath { return newCopyLocalPath(filepath.Clean(p.file)) }

func (p copyLocalPath) Join(elem copyPathSpec) copyLocalPath {
	return newCopyLocalPath(filepath.Join(p.file, elem.String()))
}

func (p copyLocalPath) Glob() ([]string, error) { return filepath.Glob(p.file) }

func copyPathIsRelative(base, target copyLocalPath) bool {
	baseNormalized := strings.ReplaceAll(stripCopyTrailingSlash(base.String()), `\`, "/")
	targetNormalized := strings.ReplaceAll(stripCopyTrailingSlash(target.String()), `\`, "/")
	relative, err := filepath.Rel(baseNormalized, targetNormalized)
	if err != nil {
		return false
	}
	return relative == "." || relative == stripCopyPathShortcuts(relative)
}

type copyRemotePath struct {
	file string
}

func newCopyRemotePath(fileName string) copyRemotePath {
	file := strings.ReplaceAll(stripCopyTrailingSlash(fileName), `\`, "/")
	path := copyRemotePath{
		file: file,
	}
	return path
}

func (p copyRemotePath) String() string { return p.file }

func (p copyRemotePath) Dir() copyRemotePath { return newCopyRemotePath(path.Dir(p.file)) }

func (p copyRemotePath) Base() copyRemotePath { return newCopyRemotePath(path.Base(p.file)) }

func (p copyRemotePath) Clean() copyRemotePath { return newCopyRemotePath(path.Clean(p.file)) }

func (p copyRemotePath) Join(elem copyPathSpec) copyRemotePath {
	return newCopyRemotePath(path.Join(p.file, elem.String()))
}

func (p copyRemotePath) StripSlashes() copyRemotePath {
	return newCopyRemotePath(stripCopyLeadingSlash(p.file))
}

func stripCopyTrailingSlash(file string) string {
	if file == "" || file == "/" {
		return file
	}
	if strings.HasSuffix(file, "/") {
		return file[:len(file)-1]
	}
	return file
}

func stripCopyLeadingSlash(file string) string { return strings.TrimLeft(file, `/\`) }

func stripCopyPathShortcuts(value string) string {
	result := value
	for {
		trimmed := strings.TrimPrefix(result, "../")
		if trimmed == result {
			break
		}
		result = trimmed
	}
	for {
		trimmed := strings.TrimPrefix(result, `..\`)
		if trimmed == result {
			break
		}
		result = trimmed
	}
	if result == "." || result == ".." {
		result = ""
	}
	if result != "" && (result[0] == '/' || result[0] == '\\') {
		return result[1:]
	}
	return result
}
