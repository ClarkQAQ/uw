// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package myers_test

import (
	"testing"

	"uw/pkg/x/tools/internal/diff/difftest"
	"uw/pkg/x/tools/internal/diff/myers"
)

func TestDiff(t *testing.T) {
	difftest.DiffTest(t, myers.ComputeEdits)
}
