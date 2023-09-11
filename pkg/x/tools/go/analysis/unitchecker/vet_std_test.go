// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package unitchecker_test

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"uw/pkg/x/tools/go/analysis/passes/asmdecl"
	"uw/pkg/x/tools/go/analysis/passes/assign"
	"uw/pkg/x/tools/go/analysis/passes/atomic"
	"uw/pkg/x/tools/go/analysis/passes/bools"
	"uw/pkg/x/tools/go/analysis/passes/buildtag"
	"uw/pkg/x/tools/go/analysis/passes/cgocall"
	"uw/pkg/x/tools/go/analysis/passes/composite"
	"uw/pkg/x/tools/go/analysis/passes/copylock"
	"uw/pkg/x/tools/go/analysis/passes/defers"
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
	"uw/pkg/x/tools/go/analysis/passes/unusedresult"
	"uw/pkg/x/tools/go/analysis/unitchecker"
)

// vet is the entrypoint of this executable when ENTRYPOINT=vet.
// Keep consistent with the actual vet in GOROOT/src/cmd/vet/main.go.
func vet() {
	unitchecker.Main(
		asmdecl.Analyzer,
		assign.Analyzer,
		atomic.Analyzer,
		bools.Analyzer,
		buildtag.Analyzer,
		cgocall.Analyzer,
		composite.Analyzer,
		copylock.Analyzer,
		defers.Analyzer,
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
		testinggoroutine.Analyzer,
		tests.Analyzer,
		timeformat.Analyzer,
		unmarshal.Analyzer,
		unreachable.Analyzer,
		// unsafeptr.Analyzer, // currently reports findings in runtime
		unusedresult.Analyzer,
	)
}

// TestVetStdlib runs the same analyzers as the actual vet over the
// standard library, using go vet and unitchecker, to ensure that
// there are no findings.
func TestVetStdlib(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in -short mode")
	}
	if version := runtime.Version(); !strings.HasPrefix(version, "devel") {
		t.Skipf("This test is only wanted on development branches where code can be easily fixed. Skipping because runtime.Version=%q.", version)
	}

	cmd := exec.Command("go", "vet", "-vettool="+os.Args[0], "std")
	cmd.Env = append(os.Environ(), "ENTRYPOINT=vet")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("go vet std failed (%v):\n%s", err, out)
	}
}
