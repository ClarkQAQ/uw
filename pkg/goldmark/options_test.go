package goldmark_test

import (
	"testing"
	"uw/pkg/goldmark/parser"
	"uw/pkg/goldmark/testutil"

	. "uw/pkg/goldmark"
)

func TestAttributeAndAutoHeadingID(t *testing.T) {
	markdown := New(
		WithParserOptions(
			parser.WithAttribute(),
			parser.WithAutoHeadingID(),
		),
	)
	testutil.DoTestCaseFile(markdown, "_test/options.txt", t, testutil.ParseCliCaseArg()...)
}
