# One Go module for the whole repo

The server, the runner, and every domain share a single `go.mod` rather than
one module per service. Multi-module repos pay for themselves when services
version independently, which nothing here does; extracting a service later is
a module path rewrite, which is cheap and mechanical.

Decided: 2026-07-24
