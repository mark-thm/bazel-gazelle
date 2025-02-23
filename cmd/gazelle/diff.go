/* Copyright 2016 The Bazel Authors. All rights reserved.

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
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/pmezard/go-difflib/difflib"
)

var errExit = fmt.Errorf("encountered changes while running diff")

func diffFile(c *config.Config, f *rule.File) error {
	rel, err := filepath.Rel(c.RepoRoot, f.Path)
	if err != nil {
		return fmt.Errorf("error getting old path for file %q: %v", f.Path, err)
	}
	rel = filepath.ToSlash(rel)

	date := "1970-01-01 00:00:00.000000000 +0000"
	diff := difflib.UnifiedDiff{
		Context:  3,
		FromDate: date,
		ToDate:   date,
	}

	newContent := f.Format()
	if bytes.Equal(newContent, f.Content) {
		// No change.
		return nil
	}

	if _, err := os.Stat(f.Path); os.IsNotExist(err) {
		diff.FromFile = "/dev/null"
	} else if err != nil {
		return fmt.Errorf("error reading original file: %v", err)
	} else if c.ReadBuildFilesDir == "" {
		diff.FromFile = rel
	} else {
		diff.FromFile = f.Path
	}

	if len(f.Content) != 0 {
		diff.A = difflib.SplitLines(string(f.Content))
	}

	diff.B = difflib.SplitLines(string(newContent))
	outPath := findOutputPath(c, f)
	if c.WriteBuildFilesDir == "" {
		diff.ToFile = rel
	} else {
		diff.ToFile = outPath
	}

	uc := getUpdateConfig(c)
	var out io.Writer = os.Stdout
	if uc.patchPath != "" {
		out = &uc.patchBuffer
	}
	if err := difflib.WriteUnifiedDiff(out, diff); err != nil {
		return fmt.Errorf("error diffing %s: %v", f.Path, err)
	}
	if ds, _ := difflib.GetUnifiedDiffString(diff); ds != "" {
		return errExit
	}

	return nil
}
