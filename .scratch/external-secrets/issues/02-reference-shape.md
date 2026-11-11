# 02 — How a value says it is a secret reference, and who sees the reference

**Status:** open
**Type:** grilling
**Blocked by:** None — can start immediately

## Question

What does a secret reference look like in a stack variable, an automation secret, a connector field and
the bootstrap secret? Decide: a URI prefix read from the value (`op://…`) or an explicit kind on the
variable; how a literal that happens to start with `op://` is kept literal; the scheme for later
managers. Then, since a reference holds no value: who may read it (stack environment masks every value
today), whether events, MCP results and the audit log may carry it, and how the web and phone apps show
a reference beside a literal.
