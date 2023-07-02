// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lostcancel_test

import (
	"uw/pkg/x/tools/go/analysis/analysistest"
	"uw/pkg/x/tools/go/analysis/passes/lostcancel"
	"uw/pkg/x/tools/internal/typeparams"
	"testing"
)

func Test(t *testing.T) {
	testdata := analysistest.TestData()
	tests := []string{"a", "b"}
	if typeparams.Enabled {
		tests = append(tests, "typeparams")
	}
	analysistest.Run(t, testdata, lostcancel.Analyzer, tests...)
}
