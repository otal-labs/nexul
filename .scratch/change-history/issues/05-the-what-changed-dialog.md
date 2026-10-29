# The "what changed" dialog

Type: prototype
Status: resolved
Blocked by: 02, 06

## Question

How the history list and the diff dialog look and behave: where the list
sits on each page, unified or side by side, raw markdown or rendered text
with highlights, how field changes (a ticket's status, a play's trigger) sit
next to a body diff, and how two versions are picked for comparison. Live
variants on the demo data, picked by watching them.

The research in 01 found the existing line diff fast enough, but prose needs
a word-level diff inside a changed line. Whether to write that or add a diff
library is part of this ticket and needs the owner's sign-off.

## Answer

- One History dialog on every page that keeps a history, opened from the
  item's menu. A header (History, the item's name and how many versions are
  kept, a "Show changes" switch, close), the body on the left, the list on the
  right, and a footer tray with "Changes since <time>" and Restore this
  version, disabled on the current one.
- The body reads as rendered text, not raw markdown. With Show changes on,
  added words are tinted green, removed words are struck through in red, and
  each changed paragraph has a short bar in its left margin: grey for edited,
  green for added, red for removed. Off, it shows the version as it was.
- Each version is compared with the one just before it. There is no picking
  two versions to compare.
- The list runs newest first: a mono time, Current on the newest, a named
  version's name, then who (several people for a sitting) and the play or
  automation it came through. It ends with a line saying only the last 10
  versions are kept, plus named ones.
- On tickets and plays, field changes sit in the same list as rows that show
  the change in place ("Status In progress → Testing"), with who and through
  what. They can't be selected; only text versions can.
- The word diff is written in the web app with no new library: the existing
  line diff compares words inside a changed paragraph, and matching
  paragraphs at the start and end are skipped before comparing, so a long doc
  with a small edit stays fast.
