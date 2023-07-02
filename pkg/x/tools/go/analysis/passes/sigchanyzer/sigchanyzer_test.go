// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sigchanyzer_test

import (
	"testing"

	"uw/pkg/x/tools/go/analysis/analysistest"
	"uw/pkg/x/tools/go/analysis/passes/sigchanyzer"
)

func Test(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.RunWithSuggestedFixes(t, testdata, sigchanyzer.Analyzer, "a")
}
