// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lsp

import (
	"context"

	"uw/pkg/x/tools/gopls/internal/lsp/mod"
	"uw/pkg/x/tools/gopls/internal/lsp/protocol"
	"uw/pkg/x/tools/gopls/internal/lsp/source"
	"uw/pkg/x/tools/internal/event"
	"uw/pkg/x/tools/internal/event/tag"
)

func (s *Server) inlayHint(ctx context.Context, params *protocol.InlayHintParams) ([]protocol.InlayHint, error) {
	ctx, done := event.Start(ctx, "lsp.Server.inlayHint", tag.URI.Of(params.TextDocument.URI))
	defer done()

	snapshot, fh, ok, release, err := s.beginFileRequest(ctx, params.TextDocument.URI, source.UnknownKind)
	defer release()
	if !ok {
		return nil, err
	}
	switch snapshot.View().FileKind(fh) {
	case source.Mod:
		return mod.InlayHint(ctx, snapshot, fh, params.Range)
	case source.Go:
		return source.InlayHint(ctx, snapshot, fh, params.Range)
	}
	return nil, nil
}
