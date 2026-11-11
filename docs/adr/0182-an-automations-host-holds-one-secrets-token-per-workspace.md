# An automations host holds one secrets-manager token per workspace, inside its supervisor

Amended: the isolation below assumed a thread could not reach what its supervisor strips from its
environment, and that failed. Every automation runs in the host's own process as the same OS user
(`nexul` on Linux) with full file access, so any of them can read every token from the unit's env
file or `/proc/self/environ`. The decision is now one workspace per token host: a host that holds a
token for a workspace runs only that workspace's automations, so an automation can only reach its own
workspace's vault, the trust ADR 0047 already accepts. `nexul automations 1password` refuses a token
for a second workspace and one for a host with another workspace's automations on it, asking the
instance as the host; the server refuses to place, create or switch on an automation on a host whose
reported tokens name another workspace, and the host runs only its token's workspace's automations.
Since every workspace's defaults run on the `instance` host, that host can hold a token only while it
serves one workspace, and a workspace created while it holds one gets its defaults switched off. Tokens are
keyed by workspace id (`NEXUL_OP_TOKEN_<id in upper case hex>`), not slug, because a rename can hand
a slug to another workspace.

Automations from several workspaces run as threads inside one host process (`worker_threads`) and
inherit its environment, and the `instance` host runs every workspace's automations. One token on the
host would let a workspace's automation read the vault items another workspace references, breaking
the isolation of ADR 0095.

Decision: a host holds one token per workspace (`nexul automations 1password <host> --workspace
<slug>`). Only the supervisor holds them, and they are stripped from the environment its threads
inherit. An automation's hello passes through the supervisor, which resolves that workspace's
references with that workspace's token and hands the values in. An automation can only ever reach its
own workspace's token, so its reach stays what ADR 0047 accepts. Handing the token to each thread
was rejected: the automation's own code would hold it, and every thread would load the SDK's 14 MB
core.

Decided: 2026-11-12. Spec: `.scratch/external-secrets/spec.md`.
