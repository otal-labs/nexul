# 06 — The template as questions

**What to build:** The server parses the Interview template into questions
per the format in 01: `##` heading per question, hint text up to the first
bullet, `- ` single-choice and `- [ ]` multi-select options with an
optional `: ` description, no bullets for free text. Saving refuses text
before the first `##`, an empty heading, or no questions, naming the line.
The 8,000-character cap leaves the template (the memory keeps it); the
template gets a 32,000-character limit. The code default becomes the twelve
questions in 02. The template's HTTP response and its MCP tool return the
parsed questions beside the body. The settings field shows the question
count instead of the length meter, at the instance and workspace layers.
"Start from the template" (`CreateInterview` copying the template into the
memory) goes: creating the interview memory starts it empty.

**Blocked by:** None — can start immediately

**Status:** resolved

- [x] Parser table tests: every rule in 01, plus an edited pre-change
      template (heading and prompt line) parsing as free-text questions
- [x] Code default is the twelve questions, parsing cleanly
- [x] HTTP and MCP return the questions; the settings field shows the count
- [x] The memory no longer starts as a copy of the template

## Answer

Built in PR_URL.
