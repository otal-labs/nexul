package main

import (
	"context"

	"github.com/otal-labs/nexul/internal/docs"
)

// collabDocWriter lets collab writes inherit docs validation without collab importing docs.
type collabDocWriter struct {
	docs *docs.Service
}

func (w collabDocWriter) CommitCollab(ctx context.Context, docID, title, body string) error {
	return w.docs.CommitCollab(ctx, docID, title, body)
}
