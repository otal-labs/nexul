# 01: How a question is written in the template

Type: grilling
Status: resolved
Blocked by: None — can start immediately

## Question

The Interview template becomes the list of questions the page steps
through, edited as markdown in the template settings like today. How is
one question written so the page can turn it into a card?

- The question text, its hint line, its options, and which option is
  recommended, if any.
- Single choice or multi-select, and whether free text is always offered.
- What happens to text the parser does not recognise, and what the save
  refuses.
- The length cap: today's 8,000 characters was sized for a memory carried
  every turn; the template no longer is one.
- How a question keeps its identity when the template is edited, so
  stored answers still line up after a question is reworded, moved, or
  deleted.

## Answer

The template stays markdown in the same settings field.

- A `##` heading is one question; the text under it, up to the first
  bullet, is the hint.
- `- ` bullets are single-choice options, `- [ ]` bullets are multi-select
  options; text after the first `: ` is the option's description. No
  bullets means free text only. Free text ("Something else…") is always
  offered.
- No recommended option in the template: it serves every project in the
  workspace, so recommendations come only from the follow-up run.
- A save is refused only for what the page cannot show: text before the
  first `##`, an empty heading, or no questions. The error names the line.
- The 8,000-character cap leaves the template and stays on the memory. The
  template gets a 32,000-character storage limit, and the settings field
  shows the question count instead of the length meter.
- A question is matched to its answer by its trimmed text. A stored answer
  keeps the question text it answered; a reworded question starts blank,
  and the orphaned answer is still handed to the follow-up run as an
  answer to an earlier question. Reordering changes nothing.
- The server parses the template; the HTTP route and the MCP tool both
  return the parsed questions alongside the body.
- Workspaces with an edited template need no migration: their heading and
  prompt-line sections parse as free-text questions with a hint.
