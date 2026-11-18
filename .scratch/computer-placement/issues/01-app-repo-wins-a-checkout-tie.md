# 01 — The app repository wins a checkout tie

**Status:** ready-for-agent
**Blocked by:** None — can start immediately

Spec: `../spec.md`, Placement, Checkout step 2. Decision 5.

`pairing.ProjectRemotes` returns each repository's remote with its role (`app` or `tests`), through
`server/cmd/wire_pairing_remotes.go`. `checkout` in `internal/pairing/placement.go` keeps today's single match, and on
several matches keeps the ones whose repository is `app`; one left wins, otherwise it stays `ambiguous_checkout`.
`CheckoutOn` and the Projects tab's checkout cell read the same answer, so a computer with the app and the tests cloned
shows its app checkout instead of a picker.

Done when: a table test in `placement_test.go` covers app plus tests (app wins), two clones of the app (ambiguous), two
tests repos with no app (ambiguous) and one match (unchanged); the Acme case from the spec lands on mini without a
pick. `paired-computers.md` says how the tie breaks.
