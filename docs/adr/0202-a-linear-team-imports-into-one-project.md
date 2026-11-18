# A Linear team imports into one project

Linear is the tracker people most often move from, and its shape differs from a board's in ways that need a call each.

Decision: the `linear` import source (ADR 0201) reads one team into one project, with a personal API key the Owner
pastes in, sent as `Authorization: <key>` with no `Bearer`, which Linear reserves for OAuth tokens.

- Files people drop into Linear are private. Rather than let the framework's downloader carry the key, every query asks
  for signed file links (`public-file-urls-expire-in`), so files still come down over plain HTTPS from
  `uploads.linear.app` alone, carrying nothing.
- A workflow state lands in a column of its own name first; failing that, its type picks the kind: triage, backlog and
  unstarted are backlog, started is progress, completed and canceled are done. A canceled issue also gets the label
  `canceled`, so it stays apart from finished work. Priority becomes a `priority:<level>` label, since tickets have no
  priority field.
- A Linear project becomes the category of the same name, and the import adds it the first time the project lacks it.
- Only `blocks` relations become blocked-by links (ADR 0159). Each issue names its relations both ways, and whichever
  end comes in second makes the link, so the order issues arrive in never loses one and a restart never adds one twice.
  Related, duplicate and similar relations have no counterpart; the scan counts them and the import skips them.
  Sub-issues come in flat, each with a "Sub-issue of KEY" line.
- Issues are read a hundred at a time with their nested labels and relations kept small enough to stay under Linear's
  10,000-point query complexity, and their comments in one query per page. The cursor after each page is the import's
  checkpoint. A rate limit is waited out in place for up to 30 seconds; a longer one pauses the import, which resumes
  from that cursor.

Rejected: OAuth (an app registration per instance for a one-time move); one project per Linear project (a team is the
unit people plan in, and its projects read naturally as swimlanes); copying related and duplicate relations as text
(nobody acts on them once moved).

Decided 2026-11-18.
