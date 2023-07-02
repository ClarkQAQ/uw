// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ignore
// +build ignore

// This file provides an example command for static checkers
// conforming to the uw/pkg/x/tools/go/analysis API.
// It serves as a model for the behavior of the cmd/vet tool in $GOROOT.
// Being based on the unitchecker driver, it must be run by go vet:
//
//	$ go build -o unitchecker main.go
//	$ go vet -vettool=unitchecker my/project/...
//
// For a checker also capable of running standalone, use multichecker.
package main

import (
	"uw/pkg/x/tools/go/analysis/unitchecker"

	"uw/pkg/x/tools/go/analysis/passes/asmdecl"
	"uw/pkg/x/tools/go/analysis/passes/assign"
	"uw/pkg/x/tools/go/analysis/passes/atomic"
	"uw/pkg/x/tools/go/analysis/passes/bools"
	"uw/pkg/x/tools/go/analysis/passes/buildtag"
	"uw/pkg/x/tools/go/analysis/passes/cgocall"
	"uw/pkg/x/tools/go/analysis/passes/composite"
	"uw/pkg/x/tools/go/analysis/passes/copylock"
	"uw/pkg/x/tools/go/analysis/passes/directive"
	"uw/pkg/x/tools/go/analysis/passes/errorsas"
	"uw/pkg/x/tools/go/analysis/passes/framepointer"
	"uw/pkg/x/tools/go/analysis/passes/httpresponse"
	"uw/pkg/x/tools/go/analysis/passes/ifaceassert"
	"uw/pkg/x/tools/go/analysis/passes/loopclosure"
	"uw/pkg/x/tools/go/analysis/passes/lostcancel"
	"uw/pkg/x/tools/go/analysis/passes/nilfunc"
	"uw/pkg/x/tools/go/analysis/passes/printf"
	"uw/pkg/x/tools/go/analysis/passes/shift"
	"uw/pkg/x/tools/go/analysis/passes/sigchanyzer"
	"uw/pkg/x/tools/go/analysis/passes/stdmethods"
	"uw/pkg/x/tools/go/analysis/passes/stringintconv"
	"uw/pkg/x/tools/go/analysis/passes/structtag"
	"uw/pkg/x/tools/go/analysis/passes/testinggoroutine"
	"uw/pkg/x/tools/go/analysis/passes/tests"
	"uw/pkg/x/tools/go/analysis/passes/timeformat"
	"uw/pkg/x/tools/go/analysis/passes/unmarshal"
	"uw/pkg/x/tools/go/analysis/passes/unreachable"
	"uw/pkg/x/tools/go/analysis/passes/unsafeptr"
	"uw/pkg/x/tools/go/analysis/passes/unusedresult"
)

func main() {
	unitchecker.Main(
		asmdecl.Analyzer,
		assign.Analyzer,
		atomic.Analyzer,
		bools.Analyzer,
		buildtag.Analyzer,
		cgocall.Analyzer,
		composite.Analyzer,
		copylock.Analyzer,
		directive.Analyzer,
		errorsas.Analyzer,
		framepointer.Analyzer,
		httpresponse.Analyzer,
		ifaceassert.Analyzer,
		loopclosure.Analyzer,
		lostcancel.Analyzer,
		nilfunc.Analyzer,
		printf.Analyzer,
		shift.Analyzer,
		sigchanyzer.Analyzer,
		stdmethods.Analyzer,
		stringintconv.Analyzer,
		structtag.Analyzer,
		tests.Analyzer,
		testinggoroutine.Analyzer,
		timeformat.Analyzer,
		unmarshal.Analyzer,
		unreachable.Analyzer,
		unsafeptr.Analyzer,
		unusedresult.Analyzer,
	)
}
