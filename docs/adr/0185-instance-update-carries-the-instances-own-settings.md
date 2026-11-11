# instance_update carries the instance's own settings

The enterprise license key (ADR 0184) is pasted and removed by the instance Owner, and agents reach every capability
the browser has (ADR 0019). `instance_get` is read-only and `instance_upgrade` starts an execution, so neither can
carry a write of a setting (`practices/mcp.md`, section 4).

Decision: one patch tool, `instance_update`, changes what belongs to the instance itself, beside `instance_get` that
reads it. Today it takes `license_key`, where a key's text replaces the current key and an empty string removes it.
It raises the tool budget by one, and the advertised size budget by the 1,000 characters it costs. The instance
settings the next enterprise features add, single sign-on and branding, become fields on it rather than tools of
their own, and `instance_get` reads them back.

Why not an existing tool: `template_update` and `workspace_update` change a template or a workspace, and a license
field there would make their arguments mean two things; `instance_upgrade` is a confirmed execution, never a setting.

Decided: 2026-11-11.
