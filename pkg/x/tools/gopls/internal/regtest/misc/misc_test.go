// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package misc

import (
	"testing"

	"uw/pkg/x/tools/gopls/internal/bug"
	"uw/pkg/x/tools/gopls/internal/hooks"
	"uw/pkg/x/tools/gopls/internal/lsp/regtest"
)

func TestMain(m *testing.M) {
	bug.PanicOnBugs = true
	regtest.Main(m, hooks.Options)
}
