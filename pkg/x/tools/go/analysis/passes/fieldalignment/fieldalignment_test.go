// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fieldalignment_test

import (
	"testing"

	"uw/pkg/x/tools/go/analysis/analysistest"
	"uw/pkg/x/tools/go/analysis/passes/fieldalignment"
)

func TestTest(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.RunWithSuggestedFixes(t, testdata, fieldalignment.Analyzer, "a")
}
