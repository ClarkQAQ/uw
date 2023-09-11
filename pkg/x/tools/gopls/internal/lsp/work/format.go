// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package work

import (
	"context"

	"uw/pkg/x/mod/modfile"
	"uw/pkg/x/tools/gopls/internal/lsp/protocol"
	"uw/pkg/x/tools/gopls/internal/lsp/source"
	"uw/pkg/x/tools/internal/event"
)

func Format(ctx context.Context, snapshot source.Snapshot, fh source.FileHandle) ([]protocol.TextEdit, error) {
	ctx, done := event.Start(ctx, "work.Format")
	defer done()

	pw, err := snapshot.ParseWork(ctx, fh)
	if err != nil {
		return nil, err
	}
	formatted := modfile.Format(pw.File.Syntax)
	// Calculate the edits to be made due to the change.
	diffs := snapshot.Options().ComputeEdits(string(pw.Mapper.Content), string(formatted))
	return source.ToProtocolEdits(pw.Mapper, diffs)
}
