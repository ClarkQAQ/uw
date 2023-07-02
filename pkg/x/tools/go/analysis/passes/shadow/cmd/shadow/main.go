// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The shadow command runs the shadow analyzer.
package main

import (
	"uw/pkg/x/tools/go/analysis/passes/shadow"
	"uw/pkg/x/tools/go/analysis/singlechecker"
)

func main() { singlechecker.Main(shadow.Analyzer) }
