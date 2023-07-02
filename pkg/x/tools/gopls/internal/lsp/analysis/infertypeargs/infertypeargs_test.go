// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package infertypeargs_test

import (
	"testing"

	"uw/pkg/x/tools/go/analysis/analysistest"
	"uw/pkg/x/tools/gopls/internal/lsp/analysis/infertypeargs"
	"uw/pkg/x/tools/internal/typeparams"
)

func Test(t *testing.T) {
	if !typeparams.Enabled {
		t.Skip("type params are not enabled")
	}
	testdata := analysistest.TestData()
	analysistest.RunWithSuggestedFixes(t, testdata, infertypeargs.Analyzer, "a")
}
