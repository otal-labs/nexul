# 04: Preserve the skipped phone acceptance verification

**Status:** wontfix

**Blocked by:** None. Scheduling verification needs the owner's decision.

## Verification gap

The shipped phone permission slice left this acceptance unchecked: an unlisted
member sees no projects, and chat works, verified through the phone Device panel.
The owner explicitly skipped this verification previously. It remains unverified;
it is not an implementation request or a reason to reopen the shipped effort.

## Evidence already available

The server side was checked on a migrated device-pass copy: `/me` sends
`projects: []` and chat permissions, and the project list is empty.
`AreaGate.test.tsx` holds the tabs. Request-count and live-event tests covered
a grant frame naming no person. These checks do not establish the phone flow.

## Scope

Retain the missing evidence and the owner's deferral. If the owner later chooses
to run it, record the phone result here. Role-aware UI ticket 14's emulator check
covers its future action sweep; it does not prove this shipped zero-project/chat
case. No code change is demanded by this ticket.

## Acceptance criteria for triage

- Keep the case explicitly unverified until it is observed on the phone.
- Record either the owner's continued deferral or the verification result.
- If verification finds a defect, triage a separately scoped implementation ticket.

## Comments

wontfix, deferred by the owner: on 2026-10-13 the owner skipped the on-phone check again ("skip it, not too fussed").
The case remains unverified on a phone; the evidence above stands as recorded. If the check is ever run, record the
result here, and triage any defect it finds as its own ticket.
