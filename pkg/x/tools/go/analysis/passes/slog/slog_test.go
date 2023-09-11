// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slog

import (
	"testing"

	"uw/pkg/x/tools/go/analysis/analysistest"
	"uw/pkg/x/tools/internal/testenv"
)

func Test(t *testing.T) {
	testenv.NeedsGo1Point(t, 21)
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, Analyzer, "a", "b")
}
