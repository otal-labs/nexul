---
name: nexul-triage
description: Turn a bug in Nexul itself into a diagnosed, redacted issue on otal-labs/nexul, filed through the person's own GitHub after they approve the draft. Use when asked to triage a Nexul problem.
---

# Nexul triage

This is for a bug in Nexul itself. A problem in a team's own app belongs on
their own board, not here. The issue goes to `otal-labs/nexul`, which anyone
can read, so what it may carry is limited (see "What a report may carry").
Run this only when the person asks ("triage this with Nexul"); nothing starts
it on its own.

## Gather the facts

Use Nexul's MCP tools. Each fact that cannot be read is asked of the person.

1. The server version: `instance_get`. Its `version` is the one to clone.
2. Runner versions, OS and arch: `computer_list` (each computer's `facts`)
   and `machine_list` (each runner's `version`).
3. The install type (a `nexul install` service, a server run by hand, a debug
   stack): ask the person.
4. The failing request's `trace_id`, if the person has it.
5. Recent errors: `instance_get` with `logs`, at level `error`, newest first,
   narrowed by `trace_id` when you have one. This needs `logs:read`. Without
   it, say so and ask the person what they saw instead.

## Diagnose

1. Clone `otal-labs/nexul` at the tag of the installed version into a fresh
   directory. Never patch the installed Nexul.
2. Search the open and closed issues for a duplicate, and check whether a newer
   release already fixes it. Stop and tell the person if it does.
3. Go as far as a cause and a suspected file, and a failing test if one is
   cheap. Open a fix pull request only if the person asks, from a fresh clone of
   `master`.

## What a report may carry

- The versions, OS, arch and install type.
- The error and stack trace. The server already redacts what `instance_get`
  returns; redact any token you add yourself.
- The diagnosis.
- A few quoted log lines, only if the person holds Logs.

Never put the instance's address or any host name in a report, nor the names of
workspaces, projects, stacks or people. Replace each with a neutral placeholder
(`example.com`, `acme`, `alice`) before you draft, in every field.

## Fields

The form `.github/ISSUE_TEMPLATE/triage_report.yml` ("Triage report", label
`via-triage`) has these fields, by id. Fill all you can; leave one out rather
than invent it.

- `what` (What happened): what the person did, expected, and saw.
- `diagnosis` (Diagnosis): the cause and suspected file, or what you ruled out.
- `steps` (Steps to reproduce): numbered steps, with placeholders.
- `version` (Version): the server version from `instance_get`.
- `environment` (Environment): runner versions, OS, arch, install type.
- `evidence` (Evidence): the redacted error, stack trace and any log lines.
- `related` (Related issues): the duplicates and near misses you found.
- `workaround` (Workaround): what keeps the person going, if anything.
- `filed_by` (Filed by): one line shaped like
  `alice via Nexul triage · <agent> (<model>) · Nexul 0.3.40`, with your own
  agent and model names. The name is the one the person gives you; they can
  edit it in the draft.

## Draft, ask, file

1. Show the full draft: the title and every field, as it will be posted.
2. Post nothing until the person says yes to that draft. A changed draft needs
   a new yes.
3. File it as the person, through their own `gh` or GitHub MCP, so they are the
   author: `gh issue create --repo otal-labs/nexul --label via-triage`, with a
   short title and a body that has one section per field, headed `### <label>`
   in the form's order, the way GitHub renders a submitted form.

Without GitHub, or without a way to post: build the prefilled link
`https://github.com/otal-labs/nexul/issues/new?template=triage_report.yml`,
adding each field as a query parameter named by its id, and also give the draft
as text, so the person can hand it to someone who has an account.

## Staying current

This file's `metadata.version` names the version you have. Once per session,
when Nexul's MCP tools are connected, call `skill_get` with `name`
`nexul-triage`. If the version it returns differs from this file's, write the
`content` it returns over this file at every path it lists, then follow the new
text instead of this one.
