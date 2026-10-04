# 09 — The two notifications

**What to build:** The two notification kinds decided in 04, with the
wording from 05: "questions asked" to the doc's watchers when a round posts
at least one question (minus its starter), and "questions answered" to the
round's starter when its last pending question is answered or skipped. Both
subject the doc, so they group under it in the inbox, and both go through
the existing open-the-doc filter. Kinds added on the server, the web inbox
labels and summaries, and the phone app's notification model and row.

**Blocked by:** 05, 07

**Status:** resolved

- [x] A watcher who cannot open the doc gets nothing; the starter is never
      told about their own round
- [x] An older phone app shows the new kinds without breaking
