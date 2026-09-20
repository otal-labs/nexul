# Search is FTS5 only; embeddings and cosine re-rank are deferred

Docs and tickets are short, keyword-shaped, and searched by people who know
roughly what they wrote. FTS5 is already in the SQLite build and needs no
model, no vector column, and no CGO, so v1 indexes lexically and stops there.
Semantic search stays a future workstream rather than a hedge built into the
schema now.

Decided: 2026-07-25
