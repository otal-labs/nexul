# An automations host holds one secrets-manager token per workspace, inside its supervisor

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
