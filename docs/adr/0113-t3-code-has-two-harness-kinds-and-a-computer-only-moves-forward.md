# T3 Code has two harness kinds, and a computer only moves forward between them

Amends ADR 0054 and ADR 0029.

T3 Code's orchestrator V2 speaks a new wire protocol, protocol 2. It ships in nightlies first, while stable T3 Code
keeps protocol 1 with no date set for its switch. A computer on protocol 1 has to keep working exactly as before, and
one whose T3 Code updates to protocol 2 has to keep working without being paired again. T3 copies its state once on
that update, so every T3 id Nexul stores stays valid across it.

Decision: protocol 2 is its own harness kind, `t3code-v2`, with its own client in `internal/t3clientv2`, beside
`t3code` and the protocol-1 client in `internal/t3client`, which stays frozen. Both clients share the transport in
`internal/t3rpc`. The protocol lives only in a computer's stored kind: there is no protocol column, and no migration,
because the kind column is free text.

The move happens in one place. The registry serves `t3code` through `harness.Forward`, which wraps the protocol-1
client with the protocol-2 one. When the protocol-1 client finds protocol 2 (T3 answers its dial with HTTP 426, or the
environment descriptor names 2), it returns `harness.MovedError`. Forward then runs the switch once and retries the
same call on the protocol-2 client. The switch is the pairing use-case's one conditional update: it moves the computer
from `t3code` to `t3code-v2` with the version the descriptor reports and writes `computer.harness_switched` in the same
transaction. Two calls racing it write one event, because the update only matches a computer still on `t3code`.
Pairing reads the descriptor before it spends the one-time `t3 pair` token, so pairing a fresh computer on a nightly
lands on `t3code-v2` with one token. A turn that loses its connection and finds T3 updated when it redials ends at once
with "T3 Code was updated during this turn; ask again"; the next mention runs on the new kind.

A computer never moves back. A `t3code-v2` computer whose T3 Code answers with protocol 1 is refused with a plain
message that names the computer and says to update T3 Code there, and a re-pair is refused the same way; re-pairing can
raise a stored kind but never lowers it. A protocol above 2 is refused as needing a newer Nexul. The refusal is
`harness.ErrProtocol`, and every surface shows its text as it is: the turn's reply, the play run, the pairing form
under the address, and the presence keeper, which keeps retrying so that updating T3 Code recovers on its own, but logs
the refusal once.

Rejected: one client that reads the protocol on every call and speaks both. It would carry protocol 1's handling into
the new client, and removing protocol 1 once T3 Code ships protocol 2 as stable would mean editing that client instead
of deleting `internal/t3client` and its registry entry. Also rejected: following a computer back to protocol 1. T3
copies its old state to the new orchestrator once, so threads created after the update exist only on the new side, and
a computer that went back would point its stored threads at a T3 Code that never saw them.

Decided 2026-10-03.
