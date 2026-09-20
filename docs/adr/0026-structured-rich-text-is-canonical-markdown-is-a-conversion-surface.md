# Structured rich text is the canonical document body; markdown is a conversion surface

A document's stored body is structured rich text (ProseMirror/Tiptap JSON),
not markdown. Markdown exists as a conversion surface at the edges: LLM tool
calls, imports, exports, and integrations read and write it, and converters
translate in both directions.

Storing markdown instead would have been the obvious choice for an
MCP-first product, but the editor needs a node tree anyway, mentions are
structured nodes rather than text that happens to parse, and CRDT
collaboration operates on that tree — keeping markdown canonical would mean
parsing and re-serializing on every keystroke and losing anything markdown
cannot express. The cost is a conversion layer that has to stay faithful in
both directions, guarded by round-trip tests; a legacy markdown body that
predates the conversion passes through unchanged.

Decided 2026-08-12.
