// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reflectvaluecompare_test

import (
	"testing"

	"uw/pkg/x/tools/go/analysis/analysistest"
	"uw/pkg/x/tools/go/analysis/passes/reflectvaluecompare"
)

func TestReflectValueCompare(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, reflectvaluecompare.Analyzer, "a")
}
