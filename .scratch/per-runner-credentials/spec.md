# Per-runner credentials

**Status:** needs-triage

> Carried over from the runner domain specification when it was dissolved.
> An old wish, not a queued build — re-confirm it is still wanted before
> slicing it.

## Problem Statement

Every runner authenticates with the same shared instance secret. An owner who
hands a colleague a runner install command, or who suspects one machine has
been compromised, has no way to cut that one runner off: revoking means
rotating the instance secret and re-enrolling every other runner by hand. The
same secret also authorises the binary download route, so it is handed out on
every enrolment and lands in shell history on every machine.

## Solution

Each runner gets its own credential instead of a copy of the instance's. An
owner can revoke a single runner from the Runners page and that machine stops
being able to connect, immediately, without any other runner noticing.
Enrolling a new runner still takes one command copied from the Add-runner
dialog.

## User Stories

1. As an owner, I want each runner to hold its own credential, so that
   revoking one machine does not disconnect the others.
2. As an owner, I want a "Revoke" action on a runner, so that a machine I no
   longer control loses access without my visiting it.
3. As an owner, I want a revoked runner's next connection attempt refused with
   a clear reason, so that I can tell revocation from a network fault when I
   read its logs.
4. As an owner, I want enrolling a runner to stay a single copied command, so
   that per-runner credentials cost me nothing at setup time.
5. As an owner, I want the credential a fresh machine enrols with to be usable
   once, or for a short window, so that a leaked install command is not a
   permanent key.
6. As an owner, I want a runner's credential to renew itself while it stays
   connected, so that short lifetimes never drop a working runner.
7. As an owner, I want the binary download route to accept a runner's own
   credential, so that enrolling a machine never requires the instance-wide
   secret.
8. As an owner, I want existing runners to keep working across the upgrade, so
   that turning this on is not a re-enrolment of every machine.
9. As an agent, I want to see which runners are revoked when I list them, so
   that I never report a revoked machine as merely offline.

## Implementation Decisions

- The credential is a stateless, server-signed token carrying the runner's
  identity, short-lived and renewed over the existing connection rather than
  stored and looked up per request — a stateless credential keeps the connect
  path free of a database read.
- Its signing key is neither the key that signs user sessions nor the existing
  runner secret, for the reason those two are already separate: the
  Add-runner dialog displays whatever the enrolment path needs, so that value
  must never be able to forge anything else.
- Revocation is a state on the runner record, checked at connect and at
  renewal. A short credential lifetime bounds how long an already-connected
  revoked runner survives; whether revocation also cuts the live connection
  immediately is a triage question.
- Enrolment — the bootstrap problem — is the open part of the design. An
  enrolment credential still mints a runner's first per-runner credential, but
  whether that enrolment credential is single-use, time-boxed, or per-machine
  needs the owner's call before anything is built.
- The shared secret stays valid for one release, so existing runners connect
  unchanged and pick up their own credential on their next connect.

## Testing Decisions

A good test asserts what an operator can observe — a connection accepted or
refused, and which runners are affected — not how a token is encoded. The seam
is the WebSocket handler's authentication path plus the runner record's
revocation state; the existing handler tests covering dispatch and
disconnection are the prior art and their fake-runner harness should drive
these. Cover at least: a valid credential connects; a revoked runner's
credential is refused; revoking one runner leaves a second connected runner
dispatchable; an expired credential renews over a live connection without
failing the job in flight; the legacy shared secret still connects during the
compatibility release.

## Out of Scope

- Per-runner authorisation (which stacks or machines a runner may deploy).
  This is authentication only; dispatch stays keyed on the machine name.
- Rotating the instance's other secrets.
- Any change to what a runner may execute once connected.

## Further Notes

Runner labels and selectors beyond the machine name were deferred alongside
this item and are deliberately not part of it: a machine scales by adding
runners to it, and selectors only start paying off once pools are
heterogeneous.
