# 29 — Structured activity and the trail as a transcript

**What to build:** The harness seam carries structured activity, not a label: kind (tool call, tool result, text, question), the tool name, a one-line argument preview, and the time; the T3 client maps T3's activity payloads onto it and the trail stores entries, not strings. The trail opens as a large centred dialog: a header with the run facts (play, starter, via, started, ended, memories, move-to, instructions) above a scrollable transcript where each step renders like the harness's own thread view: icon, tool name, argument preview, expandable detail. The thread's running line shows the latest step the same way.

**Blocked by:** None — can start immediately

**Status:** done

- [ ] `harness.Activity` carries kind, tool, summary, detail, and time; the fake harness and the T3 client populate them and a T3 wire-payload test proves the mapping
- [ ] The trail's activity column stores entries; existing string trails still read
- [ ] The trail dialog replaces the side sheet, at desktop width the transcript takes most of the viewport, and each step expands to its detail
- [ ] The live `play.run` frame carries the structured latest step
